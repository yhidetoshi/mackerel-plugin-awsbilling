[![Build Status](https://travis-ci.org/yhidetoshi/mackerel-plugin-awsbilling.svg?branch=master)](https://travis-ci.org/yhidetoshi/mackerel-plugin-awsbilling)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)

# mackerel-plugin-awsbilling


AWS billing custom metrics plugin for mackerel.io agent

## Synopsis
```
mackerel-plugin-awsbilling -region=us-east-1 -access-key-id=<id> -secret-access-key=<key>
```

## AWS IAM Policy
the credential provided manually or fetched automatically by IAM Role should have the policy that includes an action, 'cloudwatch:GetMetricStatistics'


## Example of mackerel-agent.conf
```
[plugin.metrics.awsbilling]
command = '/path/to/mackerel-plugin-awsbilling -region=us-east-1'
```
