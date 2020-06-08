package cloudwatch

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/aws/amazon-cloudwatch-agent/metric/distribution"
	"github.com/aws/amazon-cloudwatch-agent/metric/distribution/regular"
	"github.com/aws/amazon-cloudwatch-agent/metric/distribution/seh1"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const (
	maxValuesPerDatum = 5000

	// Constant for estimate encoded metric size
	// Action=PutMetricData
	pmdActionSize = 20
	// &Version=2010-08-01
	versionSize = 19
	// &MetricData.member.100.StatisticValues.Maximum=1558.3086995967291&MetricData.member.100.StatisticValues.Minimum=1558.3086995967291&MetricData.member.100.StatisticValues.SampleCount=1000&MetricData.member.100.StatisticValues.Sum=1558.3086995967291
	statisticsSize = 246
	// &MetricData.member.100.Timestamp=2018-05-29T21%3A14%3A00Z
	timestampSize = 57

	overallConstPerRequestSize = pmdActionSize + versionSize
	// &Namespace=, this is per request
	namespaceOverheads = 11

	// &MetricData.member.100.Dimensions.member.1.Name= &MetricData.member.100.Dimensions.member.1.Value=
	dimensionOverheads = 48 + 49
	// &MetricData.member.100.MetricName=
	metricNameOverheads = 34
	// &MetricData.member.100.StorageResolution=1
	highResolutionOverheads = 42
	// &MetricData.member.100.Values.member.100=1558.3086995967291 &MetricData.member.100.Counts.member.100=1000
	valuesCountsOverheads = 59 + 45
	// &MetricData.member.100.Value=1558.3086995967291
	valueOverheads = 47
	// &MetricData.member.1.Unit=Kilobytes/Second
	unitOverheads = 42
)

func publishJitter(publishInterval time.Duration) (publishJitter time.Duration) {
	hostName, _ := os.Hostname()
	jitter := computeMD5Hash(hostName, int64(publishInterval.Seconds()))
	publishJitter = time.Duration(jitter) * time.Second
	return
}

func computeMD5Hash(value string, max int64) int64 {
	md5Value := md5.New().Sum([]byte(value))
	hexString := hex.EncodeToString(md5Value)
	i, _ := strconv.ParseInt(hexString[:6], 16, 64)
	return i % max
}

func setNewDistributionFunc(maxValuesPerDatumLimit int) {
	if maxValuesPerDatumLimit >= maxValuesPerDatum {
		distribution.NewDistribution = seh1.NewSEH1Distribution
	} else {
		distribution.NewDistribution = regular.NewRegularDistribution
	}
}

func resize(dist distribution.Distribution, listMaxSize int) (distList []distribution.Distribution) {
	var ok bool
	// If this is SEH1 distribution, it has already considered the list max size.
	if _, ok = dist.(*seh1.SEH1Distribution); ok {
		distList = append(distList, dist)
		return
	}
	var regularDist *regular.RegularDistribution
	if regularDist, ok = dist.(*regular.RegularDistribution); !ok {
		log.Printf("E! The distribution type %T is not supported for resizing.", dist)
		return
	}
	values, _ := regularDist.ValuesAndCounts()
	sort.Float64s(values)
	newSEH1Dist := seh1.NewSEH1Distribution().(*seh1.SEH1Distribution)
	for i := 0; i < len(values); i++ {
		if !newSEH1Dist.CanAdd(values[i], listMaxSize) {
			distList = append(distList, newSEH1Dist)
			newSEH1Dist = seh1.NewSEH1Distribution().(*seh1.SEH1Distribution)
		}
		newSEH1Dist.AddEntry(values[i], regularDist.GetCount(values[i]))
	}
	if newSEH1Dist.Size() > 0 {
		distList = append(distList, newSEH1Dist)
	}
	return
}

func payload(datum *cloudwatch.MetricDatum) (size int) {
	size += timestampSize

	for _, dimension := range datum.Dimensions {
		size += len(*dimension.Name) + len(*dimension.Value) + dimensionOverheads
	}

	if datum.MetricName != nil {
		// The metric name won't be nil, but it should fail in the validation instead of panic here.
		size += len(*datum.MetricName) + metricNameOverheads
	}

	if datum.StorageResolution != nil {
		size += highResolutionOverheads
	}

	valuesCountsLen := len(datum.Values)
	if valuesCountsLen != 0 {
		size += valuesCountsLen*valuesCountsOverheads + statisticsSize
	} else {
		size += valueOverheads
	}

	if datum.Unit != nil {
		size += unitOverheads
	}

	return
}
