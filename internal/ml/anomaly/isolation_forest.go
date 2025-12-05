package anomaly

import (
	"fmt"
	"math"
	"math/rand"
)

// IsolationForest implements anomaly detection using Isolation Forest algorithm
type IsolationForest struct {
	trees         []*IsolationTree
	numTrees      int
	sampleSize    int
	threshold     float64
	initialized   bool
	trainingData  []float64
}

// IsolationTree represents a single isolation tree
type IsolationTree struct {
	root *IsolationNode
}

// IsolationNode represents a node in the isolation tree
type IsolationNode struct {
	splitValue  float64
	left        *IsolationNode
	right       *IsolationNode
	size        int
	isLeaf      bool
}

// NewIsolationForest creates a new isolation forest
func NewIsolationForest(numTrees, sampleSize int) *IsolationForest {
	if numTrees <= 0 {
		numTrees = 100
	}
	if sampleSize <= 0 {
		sampleSize = 256
	}

	return &IsolationForest{
		numTrees:   numTrees,
		sampleSize: sampleSize,
		threshold:  0.6, // Default anomaly threshold
		trees:      make([]*IsolationTree, numTrees),
	}
}

// Train trains the isolation forest
func (ifo *IsolationForest) Train(data []float64) error {
	if len(data) < 10 {
		return fmt.Errorf("insufficient training data: need at least 10 samples")
	}

	ifo.trainingData = make([]float64, len(data))
	copy(ifo.trainingData, data)

	// Build trees
	for i := 0; i < ifo.numTrees; i++ {
		// Sample data
		sample := ifo.sampleData(data, ifo.sampleSize)

		// Build tree
		tree := &IsolationTree{}
		tree.root = ifo.buildTree(sample, 0, maxDepth(ifo.sampleSize))
		ifo.trees[i] = tree
	}

	ifo.initialized = true
	return nil
}

// Detect checks if a value is anomalous
func (ifo *IsolationForest) Detect(value float64) (bool, float64, error) {
	if !ifo.initialized {
		return false, 0, fmt.Errorf("detector not trained")
	}

	// Calculate anomaly score
	score := ifo.anomalyScore(value)

	// Score > 0.6 typically indicates anomaly
	// Score < 0.5 typically indicates normal
	isAnomaly := score > ifo.threshold

	return isAnomaly, score, nil
}

// Update updates the detector (for isolation forest, we may retrain periodically)
func (ifo *IsolationForest) Update(value float64) error {
	if !ifo.initialized {
		return fmt.Errorf("detector not trained")
	}

	// Add to training data
	ifo.trainingData = append(ifo.trainingData, value)

	// Keep only recent data to avoid unbounded growth
	maxSize := ifo.sampleSize * 10
	if len(ifo.trainingData) > maxSize {
		ifo.trainingData = ifo.trainingData[len(ifo.trainingData)-maxSize:]
	}

	return nil
}

// Reset resets the detector
func (ifo *IsolationForest) Reset() {
	ifo.trees = make([]*IsolationTree, ifo.numTrees)
	ifo.initialized = false
	ifo.trainingData = nil
}

// buildTree recursively builds an isolation tree
func (ifo *IsolationForest) buildTree(data []float64, depth, maxDepth int) *IsolationNode {
	node := &IsolationNode{
		size: len(data),
	}

	// Stop conditions
	if len(data) <= 1 || depth >= maxDepth {
		node.isLeaf = true
		return node
	}

	// Find min and max
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// If all values are the same
	if min == max {
		node.isLeaf = true
		return node
	}

	// Random split point
	node.splitValue = min + rand.Float64()*(max-min)

	// Split data
	var left, right []float64
	for _, v := range data {
		if v < node.splitValue {
			left = append(left, v)
		} else {
			right = append(right, v)
		}
	}

	// Build child nodes
	if len(left) > 0 {
		node.left = ifo.buildTree(left, depth+1, maxDepth)
	}
	if len(right) > 0 {
		node.right = ifo.buildTree(right, depth+1, maxDepth)
	}

	return node
}

// pathLength calculates the path length for a value in a tree
func (ifo *IsolationForest) pathLength(value float64, node *IsolationNode, depth int) float64 {
	if node == nil || node.isLeaf {
		// Add average path length for unseen data
		return float64(depth) + avgPathLength(node.size)
	}

	if value < node.splitValue && node.left != nil {
		return ifo.pathLength(value, node.left, depth+1)
	}

	if node.right != nil {
		return ifo.pathLength(value, node.right, depth+1)
	}

	return float64(depth) + avgPathLength(node.size)
}

// anomalyScore calculates the anomaly score for a value
func (ifo *IsolationForest) anomalyScore(value float64) float64 {
	if len(ifo.trees) == 0 {
		return 0
	}

	var avgPathLen float64
	for _, tree := range ifo.trees {
		avgPathLen += ifo.pathLength(value, tree.root, 0)
	}
	avgPathLen /= float64(len(ifo.trees))

	// Normalize: 2^(-avgPathLen / c)
	// c is the average path length of unsuccessful search in BST
	c := avgPathLength(ifo.sampleSize)
	score := math.Pow(2, -avgPathLen/c)

	return score
}

// sampleData randomly samples data
func (ifo *IsolationForest) sampleData(data []float64, sampleSize int) []float64 {
	if len(data) <= sampleSize {
		result := make([]float64, len(data))
		copy(result, data)
		return result
	}

	sample := make([]float64, sampleSize)
	indices := rand.Perm(len(data))
	for i := 0; i < sampleSize; i++ {
		sample[i] = data[indices[i]]
	}

	return sample
}

// maxDepth calculates maximum tree depth
func maxDepth(sampleSize int) int {
	return int(math.Ceil(math.Log2(float64(sampleSize))))
}

// avgPathLength calculates average path length for a given size
func avgPathLength(size int) float64 {
	if size <= 1 {
		return 0
	}
	if size == 2 {
		return 1
	}

	// H(size-1) - (size-1)/size where H is harmonic number
	h := harmonicNumber(size - 1)
	return 2*h - 2*float64(size-1)/float64(size)
}

// harmonicNumber calculates the harmonic number
func harmonicNumber(n int) float64 {
	if n <= 0 {
		return 0
	}
	// Approximation: H(n) H ln(n) + 0.5772156649 (Euler-Mascheroni constant)
	return math.Log(float64(n)) + 0.5772156649
}
