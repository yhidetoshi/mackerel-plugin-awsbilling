# mackerel-plugin-awsbilling
AWS billing custom metrics plugin for mackerel.io agent

## Synopsis
```
mkr-plugin-aws-billing -region=us-east-1 -access-key-id=<id> -secret-access-key=<key>
```

## AWS IAM Policy
the credential provided manually or fetched automatically by IAM Role should have the policy that includes an action, 'cloudwatch:GetMetricStatistics'


## Example of mackerel-agent.conf
```
[plugin.metrics.awsbilling]
command = '/path/to/mkr-plugin-aws-billing -region us-east-1'
```