package worker

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"gitlab.tubit.tu-berlin.de/dominik-ernst/tracer-benchmarks/api"
)

func init() {
	distributionRegistry = make(map[string]DistributionSampler)
	distributionRegistry["constant"] = &StaticDistribution{}
	//rng := rand.New(rand.NewSource(time.Now().UnixNano()))
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
const (
	letterIdxBits = 7                    // 7 bits to represent a letter index, because we have 26*2 + 10
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

//RandStringWithLength implements random string generation based on https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func RandStringWithLength(n int64) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

var distributionRegistry map[string]DistributionSampler

type DistributionIndex struct {
	Name       string
	ParamNames []string
	Parameters map[string]float64
	RNG        *rand.Rand
	Wrapper    DistributionSampler
}
type StaticDistribution struct {
	Value int64
}

type NoDistribution struct {
}

func (sd *NoDistribution) SetParameters(values map[string]float64) {
	//do nothing
}

func (sd *NoDistribution) GetNextValue() time.Duration {
	return time.Duration(0)
}

func (sd *NoDistribution) SetRNGSeed(seed int64) {
	//do nothing
}

func (sd *StaticDistribution) SetParameters(values map[string]float64) {
	sd.Value = int64(values["value"])
}

func (sd *StaticDistribution) GetNextValue() time.Duration {
	return time.Duration(sd.Value) * time.Microsecond
}

func (sd *StaticDistribution) SetRNGSeed(seed int64) {
	//do nothing
}

type DistributionSampler interface {
	SetParameters(map[string]float64)
	GetNextValue() time.Duration
	SetRNGSeed(int64)
}

func LookupDistribution(work *api.Work) (DistributionSampler, error) {
	if work == nil {
		return &NoDistribution{}, nil
	}
	if strings.Compare("none", strings.ToLower(work.DistType)) == 0 {
		return &NoDistribution{}, nil
	}
	dist, exists := distributionRegistry[work.DistType]
	if !exists {
		return nil, errors.New("Unknown distribution")
	}
	dist.SetParameters(work.Parameters)
	//TODO check if params are valid for distribution
	return dist, nil
}

type GaussianWork struct {
	Mean   float64 `yaml:"mean"`
	StdDev float64 `yaml:"stddev"`
}

func (w *GaussianWork) GetNextValue() int64 {
	return int64(rand.NormFloat64()*w.StdDev + w.Mean)
}
