package mpawsbilling

import (
	"flag"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

const (
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


func (p *AwsBillingPlugin) prepare() error {
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
	statsInput := make([]*string, len(metric.Metrics))
	for i, typ := range metric.Metrics {
		statsInput[i] = aws.String(typ.Type)
	}
	input := &cloudwatch.GetMetricStatisticsInput{
		StartTime:  aws.Time(time.Now().Add(time.Hour * -24)),
		EndTime:    aws.Time(time.Now()),
		MetricName: aws.String("EstimatedCharges"),
		Period:     aws.Int64(86400),
		Statistics: []*string{
			aws.String(cloudwatch.StatisticMaximum),
		},
		Namespace: aws.String("AWS/Billing"),
	}
	input.Dimensions = []*cloudwatch.Dimension{
		{
			Name:  aws.String("Currency"),
			Value: aws.String("USD"),
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

func (p AwsBillingPlugin) FetchMetrics() (map[string]float64, error) {
	stats := make(map[string]float64)
	for _, met := range AwsGroup {
		v, err := getLastPointFromCloudWatch(p.CloudWatch,met)
		if err != nil {
			err.Error()
		}
		if v != nil {
			stats = mergeStatsFromDatapoint(stats, v, met)
		}
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
	{CloudWatchName: "EstimatedCharges", Metrics: []metric{
		{MackerelName: "EstimatedCharges", Type: metricsTypeMaximum},
	}},
}

func (p AwsBillingPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := p.MetricsLabelPrefix()

	graphdef := map[string]mp.Graphs{
		"requests": {
			Label: labelPrefix,
			Unit:  mp.UnitFloat,
			Metrics: []mp.Metrics{
				{Name: "EstimatedCharges", Label: "EstimatedCharges", Stacked: true},
			},
		},
	}
	return graphdef
}


// Do the plugin
func Do() {
	optAccessKeyID := flag.String("access-key-id", "", "AWS Access Key ID")
	optSecretAccessKey := flag.String("secret-access-key", "", "AWS Secret Access Key")
	optRegion := flag.String("region", "", "AWS Region")
	flag.Parse()

	var plugin AwsBillingPlugin
	plugin.AccessKeyID = *optAccessKeyID
	plugin.SecretAccessKey = *optSecretAccessKey
	plugin.Region = *optRegion

	err := plugin.prepare()
	if err != nil{
		log.Fatal(err)
	}

	mp.NewMackerelPlugin(plugin).Run()
}