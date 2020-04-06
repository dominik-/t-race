package worker

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/dominik-/t-race/api"
)

var src = rand.NewSource(time.Now().UnixNano())

func init() {
	distributionRegistry = make(map[string]DistributionSampler)
	distributionRegistry["constant"] = &StaticDistribution{}
	distributionRegistry["gaussian"] = &GaussianDistribution{
		randomizer: rand.New(src),
	}
	distributionRegistry["exponential"] = &ExpDistribution{
		randomizer: rand.New(src),
	}
	//rng := rand.New(rand.NewSource(time.Now().UnixNano()))
}

//Alphabet for random key/value generation from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
const (
	letterIdxBits = 7                    // 7 bits to represent a letter index, because we have 26*2 + 10
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

//RandStringWithLength implements random string generation; based on https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
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

type NoDistribution struct {
}

func (nd *NoDistribution) SetParameters(values map[string]float64) {
	//do nothing
}

func (nd *NoDistribution) GetNextValue() time.Duration {
	return time.Duration(0)
}

func (nd *NoDistribution) SetRNGSeed(seed int64) {
	//do nothing
}

type StaticDistribution struct {
	Value int64
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

// DistributionSampler describes a statistic distribution and allows setting parameters (based on YAML configs) and a seed for that distribution.
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

type GaussianDistribution struct {
	randomizer *rand.Rand
	mean       float64 `yaml:"mean"`
	stdDev     float64 `yaml:"stddev"`
}

func (gd *GaussianDistribution) GetNextValue() time.Duration {
	return time.Duration(gd.randomizer.NormFloat64()*gd.stdDev+gd.mean) * time.Microsecond
}

func (gd *GaussianDistribution) SetParameters(values map[string]float64) {
	//TODO should probably add some parsing/validation
	gd.mean = values["mean"]
	gd.stdDev = values["stddev"]
}

func (gd *GaussianDistribution) SetRNGSeed(seed int64) {
	gd.randomizer = rand.New(rand.NewSource(seed))
}

type ExpDistribution struct {
	randomizer *rand.Rand
	mean       float64 `yaml:"mean"`
	lambda     float64 `yaml:"-"`
}

func (ed *ExpDistribution) SetRNGSeed(seed int64) {
	ed.randomizer = rand.New(rand.NewSource(seed))
}

func (ed *ExpDistribution) SetParameters(values map[string]float64) {
	ed.mean = values["mean"]
	ed.lambda = 1 / ed.mean
}

func (ed *ExpDistribution) GetNextValue() time.Duration {
	return time.Duration(ed.randomizer.ExpFloat64()/ed.lambda) * time.Microsecond
}
