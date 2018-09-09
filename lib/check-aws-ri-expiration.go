package ri_expiration

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	"github.com/mackerelio/golib/logging"
)

var logger = logging.GetLogger("metrics.plugin.aws-ri-expiration")

// AwsRiExpirationPlugin stores the parameters for aws-ri-expiration Mackerel plugin
type AwsRiExpirationPlugin struct {
	Prefix            string
	ReservedInstances []purchasedReservedInstance
}

// MetricKeyPrefix returns the metrics key prefix
func (q AwsRiExpirationPlugin) MetricKeyPrefix() string {
	if q.Prefix == "" {
		q.Prefix = "aws-ri-expiration"
	}
	return q.Prefix
}

// FetchMetrics interface for mackerelplugin
func (q AwsRiExpirationPlugin) FetchMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	for _, ri := range q.ReservedInstances {
		// t3_small_1years_494USD_3instances: 355
		key := ri.key()
		metrics[key] = uint64(ri.DaysLeft)
	}
	return metrics, nil

}

// GraphDefinition interface for mackerelplugin
func (q AwsRiExpirationPlugin) GraphDefinition() map[string]mp.Graphs {

	labelPrefix := strings.Title(q.MetricKeyPrefix())

	var metricsDef []mp.Metrics
	for _, ri := range q.ReservedInstances {
		key := ri.key()
		metricsDef = append(metricsDef,
			mp.Metrics{Name: key, Label: key, Type: "uint64", Diff: false, Stacked: false})
	}
	var graphdef = map[string]mp.Graphs{
		"": {
			Label:   (labelPrefix + "Days Left"),
			Unit:    "integer",
			Metrics: metricsDef,
		},
	}
	return graphdef
}

// Do the plugin
func Do() {
	optAWSRegion := flag.String("region", "", "AWS Region")
	optPrefix := flag.String("metric-key-prefix", "aws-ri-expiration", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(*optAWSRegion),
	}))

	var pris []purchasedReservedInstance

	ec2Client := ec2.New(sess)
	ris, err := getReservedEC2Instances(ec2Client)
	if err != nil {
		logger.Errorf("getReservedEC2Instances failed: %s", err)
	}
	pris = append(pris, ris...)

	rdsClient := rds.New(sess)
	ris, err = getReservedRDSInstances(rdsClient)
	if err != nil {
		logger.Errorf("getReservedRDSInstances failed: %s", err)
	}
	pris = append(pris, ris...)

	helper := mp.NewMackerelPlugin(AwsRiExpirationPlugin{
		Prefix:            *optPrefix,
		ReservedInstances: pris,
	})
	helper.Tempfile = *optTempfile
	helper.Run()
}

type purchasedReservedInstance struct {
	InstanceType       string
	InstanceCount      int64
	FixedPrice         float64
	CurrencyCode       string
	ReservedInstanceID string
	DaysLeft           int64
}

func (p purchasedReservedInstance) key() string {
	return fmt.Sprintf("%s_%d%s_%dinstances_%s",
		strings.Replace(p.InstanceType, ".", "_", -1), int(p.FixedPrice), p.CurrencyCode, p.InstanceCount, p.ReservedInstanceID)
}

func getReservedEC2Instances(client *ec2.EC2) ([]purchasedReservedInstance, error) {
	o, err := client.DescribeReservedInstances(nil)
	if err != nil {
		return nil, err
	}

	utcNow := time.Now().UTC()
	var pris []purchasedReservedInstance
	for _, ri := range o.ReservedInstances {
		daysLeft := int64((*ri.End).Sub(utcNow).Hours() / 24)
		pris = append(pris, purchasedReservedInstance{
			InstanceType:       *ri.InstanceType,
			InstanceCount:      *ri.InstanceCount,
			FixedPrice:         *ri.FixedPrice,
			CurrencyCode:       *ri.CurrencyCode,
			ReservedInstanceID: *ri.ReservedInstancesId,
			DaysLeft:           daysLeft,
		})
	}
	return pris, nil
}

func getReservedRDSInstances(client *rds.RDS) ([]purchasedReservedInstance, error) {
	o, err := client.DescribeReservedDBInstances(nil)
	if err != nil {
		return nil, err
	}

	utcNow := time.Now().UTC()
	var pris []purchasedReservedInstance
	for _, ri := range o.ReservedDBInstances {
		endTime := ri.StartTime.Add(time.Duration(*ri.Duration) * time.Second)
		daysLeft := int64(endTime.Sub(utcNow).Hours() / 24)
		pris = append(pris, purchasedReservedInstance{
			InstanceType:       *ri.DBInstanceClass,
			InstanceCount:      *ri.DBInstanceCount,
			FixedPrice:         *ri.FixedPrice,
			CurrencyCode:       *ri.CurrencyCode,
			ReservedInstanceID: *ri.ReservedDBInstancesOfferingId,
			DaysLeft:           daysLeft,
		})
	}
	return pris, nil
}
