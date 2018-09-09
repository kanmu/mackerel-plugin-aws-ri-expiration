# mackerel-plugin-aws-ri-expiration

This is a custom metrics plugin for [mackerel.io](https://mackerel.io/) agent, which reports the number of days left until reserved instances expire.

## Synopsis

```shell
mackerel-plugin-aws-ri-expiration [-region=<aws-region>]
                                  [-metric-key-prefix=<prefix>]
```

## Example of mackerel-agent.conf

```toml
[plugin.metrics.aws-ri-expiration]
command = [
    "/usr/local/bin/mackerel-plugin-aws-ri-expiration",
    "--region=ap-northeast-1",
]
```

## Support

- [X] EC2
- [X] RDS
- [ ] ElastiCache
- [ ] Redshift

## License

MIT License.
