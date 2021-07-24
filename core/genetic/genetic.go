package genetic

// Genetic is an object in charge of performing the basic functions of a genetic algorithm
type Genetic interface {
	Mutate(c *Chromosome, rate float64)
	Fitness(population []*Chromosome) error
	Selection(population []*Chromosome, size int) ([]*Chromosome, error)
	Genesis(c *Chromosome) map[string][]*Gene
}

// Chromosome is a tree representation of an especific targeting configuration with some
// information relevant to a genetic algorithm
type Chromosome struct {
	ID      string
	Root    *Gene
	Fitness float64
	Quality float64
}

// Gene is used to configure the result of targeting required
// by the caller
type Gene struct {
	ID       string
	Name     string
	Type     string
	Value    float64
	Parent   *Gene
	Children []*Gene
}

// Insert, Search and Delete
