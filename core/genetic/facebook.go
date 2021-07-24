package genetic

import (
	"errors"
	"sort"
	"sync"
)

type facebook struct {
	distribution func() float64
	quality      func(c *Chromosome) (float64, error)
	fitness      func(population []*Chromosome, q ...float64) error
	selection    func(population []*Chromosome, size int, d func() float64, fitness func(population []*Chromosome, q ...float64) error) ([]*Chromosome, error)
}

var (
	// ErrorInvalidQualityStandardFitness the quality provided is worng
	ErrorInvalidQualityStandardFitness = errors.New("the value of the quality parameter for the standard fitness is wrong")
	// ErrorInvalidPopulation the provided population for the standard selection is not large enought to perform the selection operation
	ErrorInvalidPopulation = errors.New("The population size can't be less than or equal to selection size")
)

// New initialices standard fitness and standard selection genetic algorithm
// to use with facebook data
func New(quality func(c *Chromosome) (float64, error)) Genetic {
	return &facebook{
		distribution: generateRandom,
		quality:      quality,
		fitness:      standardFitness,
		selection:    standardSelection,
	}
}

func (f *facebook) Genesis(c *Chromosome) map[string][]*Gene {
	var result = make(map[string][]*Gene)
	var queue = []*Gene{c.Root}

	for {
		if len(queue) == 1 && queue[0] != c.Root {
			if queue[0].ID != "" && queue[0].Value == 1 {
				result[queue[0].Type] = append(result[queue[0].Type], queue[0])
			}
			break
		}
		if queue[0].ID != "" {
			if queue[0].Value == 1 {
				result[queue[0].Type] = append(result[queue[0].Type], queue[0])
			}
		}
		queue = append(queue, queue[0].Children...)
		queue = queue[1:]
	}

	return result
}

func (f *facebook) Mutate(c *Chromosome, rate float64) {
	var wg sync.WaitGroup
	go func(wg *sync.WaitGroup) {
		wg.Add(1)
		binaryMutation(c.Root, f.distribution, rate, wg)
		wg.Done()
	}(&wg)

	wg.Wait()
}

func binaryMutation(gene *Gene, d func() float64, rate float64, wg *sync.WaitGroup) {
	// In the facebook Graph only leaf nodes have an id and therefore
	// the function can return after finding a leave node.
	if gene.ID != "" {
		if d() <= rate {
			if gene.Value == 1 {
				gene.Value = 0
			} else {
				gene.Value = 1
			}
		}
	}
	for i := 0; i < len(gene.Children); i++ {
		go func(wg *sync.WaitGroup, i int) {
			wg.Add(1)
			binaryMutation(gene.Children[i], d, rate, wg)
			wg.Done()
		}(wg, i)
	}
}

func (f *facebook) Fitness(population []*Chromosome) error {
	var q float64
	for i := 0; i < len(population); i++ {
		qi, err := f.quality(population[i])
		if err != nil {
			return err
		}
		population[i].Quality = qi
		q += qi
	}

	return f.fitness(population, q)
}

// standardFitness computes the fitness of a chromosome relative to the overall fitness of the population
func standardFitness(population []*Chromosome, q ...float64) error {
	if len(q) != 1 {
		return ErrorInvalidQualityStandardFitness
	}
	for i := 0; i < len(population); i++ {
		population[i].Fitness = population[i].Quality / q[0]
	}

	return nil
}

func (f *facebook) Selection(population []*Chromosome, size int) ([]*Chromosome, error) {
	return f.selection(population, size, f.distribution, standardFitness)
}

func standardSelection(population []*Chromosome, size int, d func() float64, fitness func(population []*Chromosome, q ...float64) error) ([]*Chromosome, error) {
	if len(population) == size {
		return population, nil
	}
	if len(population) < size {
		return nil, ErrorInvalidPopulation
	}

	selected := make([]*Chromosome, size)

	for i := 0; i < len(selected); i++ {
		f := cumulative(population)
		idx := sort.SearchFloat64s(f, d())
		selected[i] = population[idx]
		if idx == len(population) {
			population = population[:idx-1]
		} else {
			population = append(population[:idx], population[idx+1:]...)
		}
		err := fitness(population, sumQuality(population))
		if err != nil {
			return nil, err
		}
	}

	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].Quality > selected[j].Quality
	})

	return selected, nil
}

func cumulative(population []*Chromosome) []float64 {
	var a float64

	d := make([]float64, len(population))
	for i := 0; i < len(population)-1; i++ {
		a += population[i].Fitness
		d[i] = a
	}
	// Fix for floating point error
	d[len(d)-1] = 1

	return d
}

func sumQuality(population []*Chromosome) float64 {
	var q float64
	for i := 0; i < len(population); i++ {
		q += population[i].Quality
	}

	return q
}
