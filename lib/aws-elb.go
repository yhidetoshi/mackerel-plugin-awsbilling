package mpawsbilling

import (
	"flag"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

const (
	namespace 		   = "AWS/Billing"
	region 			   = "us-east-1"
	metricsName  	   = "EstimatedCharges"
	metricsTypeMaximum = "Maximum"
)

type metricsGroup struct{
	CloudWatchName string
	Metrics 	   []metric
}


type metric struct{
	MackerelName string
	Type string
}

type AwsBillingPlugin struct{
	Name			string
	Region 			string
	AccessKeyID		string
	SecretAccessKey string
	LabelPrefix		string
	CloudWatch		*cloudwatch.CloudWatch
}

func (p AwsBillingPlugin) MetricsLabelPrefix() string{
	if p.LabelPrefix == ""{
		return "AWS/Billing"
	}
	return p.LabelPrefix
}


func (p AwsBillingPlugin) prepare() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	config := aws.NewConfig()
	if p.AccessKeyID != "" && p.SecretAccessKey != "" {
		config = config.WithCredentials(credentials.NewStaticCredentials(p.AccessKeyID, p.SecretAccessKey, ""))
	}
	if p.Region != "" {
		config = config.WithRegion(p.Region)
	}

	p.CloudWatch = cloudwatch.New(sess, config)

	return nil
}

func getLastPointFromCloudWatch(cw cloudwatchiface.CloudWatchAPI, metric metricsGroup) (*cloudwatch.Datapoint, error) {
	now := time.Now()
	statsInput := make([]*string, len(metric.Metrics))
	for i, typ := range metric.Metrics {
		statsInput[i] = aws.String(typ.Type)
	}
	input := &cloudwatch.GetMetricStatisticsInput{
		StartTime:  aws.Time(now.Add(time.Duration(300) * time.Second * -1)), // 5 min
		EndTime:    aws.Time(now),
		MetricName: aws.String(metric.CloudWatchName),
		Period:     aws.Int64(60),
		Statistics: statsInput,
		Namespace:  aws.String(namespace),
	}
	input.Dimensions = []*cloudwatch.Dimension{
		{
			Name:  aws.String("BucketName"),
			Value: aws.String(bucketName),
		},
		{
			Name:  aws.String("FilterId"),
			Value: aws.String(filterID),
		},
	}
	response, err := cw.GetMetricStatistics(input)
	if err != nil {
		return nil, err
	}

	datapoints := response.Datapoints
	if len(datapoints) == 0 {
		return nil, nil
	}

	latest := new(time.Time)
	var latestDp *cloudwatch.Datapoint
	for _, dp := range datapoints {
		if dp.Timestamp.Before(*latest) {
			continue
		}

		latest = dp.Timestamp
		latestDp = dp
	}

	return latestDp, nil
}

// FetchMetrics fetch elb metrics
func (p AwsBillingPlugin) FetchMetrics() (map[string]float64, error) {
	stats := make(map[string]float64)
	v, err := getLastPointFromCloudWatch(p.CloudWatch)
	if err != nil{
		err.Error()
	}
	if v != nil {
		stats = mergeStatsFromDatapoint(stats, err)
	}
	return stats, nil
}

func mergeStatsFromDatapoint(stats map[string]float64, dp *cloudwatch.Datapoint, mg metricsGroup) map[string]float64 {
	for _, met := range mg.Metrics {
		switch met.Type {
		case metricsTypeMaximum:
			stats[met.MackerelName] = *dp.Maximum
		}
	}
	return stats
}

var AwsGroup = []metricsGroup{
	{CloudWatchName: "AllRequests", Metrics: []metric{
		{MackerelName: "AllRequests", Type: metricsTypeSum},
	}},
}

func (p AwsBillingPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := p.MetricsLabelPrefix()

	graphdef := map[string]mp.Graphs{
		"requests": {
			Label: labelPrefix,
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "EstimatedCharges", Label: "EstimatedCharges", Stacked: "true"},
			},
		},
	}
	return graphdef
}


// Do the plugin
func Do() {
	optAccessKeyID := flag.String("access-key-id", "", "AWS Access Key ID")
	optSecretAccessKey := flag.String("secret-access-key", "", "AWS Secret Access Key")
	flag.Parse()

	var plugin AwsBillingPlugin
	plugin.AccessKeyID = *optAccessKeyID
	plugin.SecretAccessKey = *optSecretAccessKey

	err := plugin.prepare()
	if err != nil{
		log.Fatal(err)
	}

	mp.NewMackerelPlugin(plugin).Run()
}