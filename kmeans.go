// kmeans assigns clusters to points using a k-means calculation
//   - initial centroids are generated randomly
//   - each point is assigned to its closest centroid
//   - new centroids are calculated as the mean of their points
//   - second and third steps repeat until centroids converge
//   - convergence defined as consecutive (nearly) identical centroid
//     sets: the squared Euclidean distance between corresponding
//     centroids in the old and new sets is less than the given
//     threshold
//
// the program does not save the cluster assignments anywhere as correctness
// is judged by the final set of centroids
//
// points are read in from an input file; the file format and the example
// input files used are directly from the University of Texas at Austin
// CS380P online Parallel Systems course K-means clustering lab
// https://www.cs.utexas.edu/~rossbach/cs380p/index.html
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CentroidAssignment struct {
	point    []float64
	centroid int
}

func main() {
	var inputFileName = ""
	flag.StringVar(&inputFileName, "input", "", "path to input file (required)")
	var threshold = 0.0
	flag.Float64Var(&threshold, "threshold", 0.0, "convergence threshold (required)")
	var numClusters = 0
	flag.IntVar(&numClusters, "clusters", 0, "number of clusters (required)")
	var numWorkers = 0
	flag.IntVar(&numWorkers, "workers", 0, "number of workers (goroutines) to calculate point assignments (required)")
	var randSeed uint64 = 0
	flag.Uint64Var(&randSeed, "seed", 0, "seed for rndom number generation (optional; defaults to 0)")
	var outputCentroids = false
	flag.BoolVar(&outputCentroids, "centroids", false, "whether to output the final centroids")
	flag.Parse()
	if inputFileName == "" || numClusters == 0 || numWorkers == 0 || threshold == 0.0 {
		flag.Usage()
		return
	}

	points := readPointsFile(inputFileName)
	numPoints := len(*points)
	numDimensions := len((*points)[0])
	if numWorkers > numPoints {
		log.Fatalf("The number of workers (%d) should not exceed the number of points (%d)", numWorkers, numPoints)
	}

	// now that the data has been imported, begin timing the clustering calculation
	startTime := time.Now()

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(numWorkers)
	centroidsChan := make(chan *[][]float64)
	continueChan := make(chan bool)
	// the channel for centroid assignment can buffer up to all of the assignments
	centroidAssignmentChan := make(chan CentroidAssignment, numPoints)
	centroids := firstCentroids(numClusters, points, randSeed)

	for blockNumber := range numWorkers {
		startIndex, stopIndex := getIndexes(numPoints, numWorkers, blockNumber)
		go pointsWorker((*points)[startIndex:stopIndex], centroidsChan, continueChan, centroidAssignmentChan, &waitGroup)
	}

	iterations := 0
	for {
		iterations++
		newCentroids := contiguous2DFloats(numClusters, numDimensions)
		assignments := make([]int, numClusters)
		// signal all workers to assign their points
		for range numWorkers {
			centroidsChan <- centroids
		}

		// calculate new centroids based on the assignments
		// NOTE more concurrency could be achieved by spawning goroutines,
		// up to the number of clusters, to calculate the new centroids;
		// would this require each pointsWorker to have an array of channels,
		// one per centroid, and ship each point assignment off to the
		// correct centroid channel? That, or the main thread here could still
		// collect all assignments and shunt them off to the correct centroid
		// channel itself...
		for range numPoints {
			nextAssignment := <-centroidAssignmentChan
			// for the assigned centroid, increase its assignment count
			assignments[nextAssignment.centroid]++
			// accumulate the dimensions of this latest assigned point, for later averaging
			assignedCentroid := (*newCentroids)[nextAssignment.centroid]
			for dimIndex, dim := range nextAssignment.point {
				assignedCentroid[dimIndex] += dim
			}
		}
		// all dimensions have been accumulated to new centroids for all points;
		// average the dimensions to finalize new centroid calculations
		for centIndex, centroid := range *newCentroids {
			divisor := assignments[centIndex]
			for dimIndex := 0; dimIndex < numDimensions; dimIndex++ {
				centroid[dimIndex] /= float64(divisor)
			}
		}

		if vectorsConverged(*centroids, *newCentroids, threshold) {
			for range numWorkers {
				continueChan <- false
			}
			break
		}

		// replace current centroids with new centroids for next iteration
		// of clustering
		centroids = newCentroids
		// signal all the pointWorkers to run another iteration
		for range numWorkers {
			continueChan <- true
		}
	}

	waitGroup.Wait()

	elapsedTime := time.Since(startTime)
	fmt.Println("Total time (ms): ", elapsedTime.Milliseconds())
	fmt.Println("Iterations to convergence: ", iterations, "; time per iteration (ms): ", float64(elapsedTime.Milliseconds())/float64(iterations))
	if outputCentroids {
		for centIndex, centroid := range *centroids {
			fmt.Printf("%d", centIndex)
			for _, dim := range centroid {
				fmt.Printf(" %f", dim)
			}
			fmt.Println()
		}
	}
}

// pointsWorker receives a slice of points to work on
//   - don't start working until receiving netroids on the centroidsChan channel
//   - for each point in the slice, find the centroid it's closest to and
//     report back via the assignment channel
//   - when all points are done wait for a message on the continueChan channel
//   - if the continueChan message is false then exit
//   - if the continueChan message is true then wait on the centroidsChan channel again
//
// two channel signals (continueChan and centroidsChan) are necessary because if the only
// signal was centroidsChan then a speedy goroutine could steal the centroidsChan signal meant
// for some other worker goroutine, causing that other goroutine to starve for work;
// meanwhile the stealing routine would start more calculations prematurely
//
// assumes points and centroids all have the same number of dimensions
func pointsWorker(points [][]float64, centroidsChan chan *[][]float64, continueChan chan bool, centroidAssignmentChan chan CentroidAssignment, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for {
		centroids := <-centroidsChan
		for _, point := range points {
			centAssignment := CentroidAssignment{point, -1}
			minimumSquaredDistance := math.MaxFloat64
			for centIndex, cent := range *centroids {
				nextSquaredDistance := squareOfEuclideanDistance(point, cent)
				if nextSquaredDistance < minimumSquaredDistance {
					minimumSquaredDistance = nextSquaredDistance
					centAssignment.centroid = centIndex
				}
			}
			centroidAssignmentChan <- centAssignment
		}
		if !<-continueChan {
			return
		}
	}
}

// readPointsFile reads in points from the nnamed input file
//
// assumes the file is (mostly) well-formed; specifically that
// it has the number of points it claims to, and that each point
// has the same number of dimensions
func readPointsFile(inputFileName string) *[][]float64 {
	var points *[][]float64 = nil
	numDimensions := 0

	inputFile, error := os.Open(inputFileName)
	if error != nil {
		log.Fatal(error)
	}
	defer inputFile.Close()

	inputScanner := bufio.NewScanner(inputFile)

	// the first line is the number of points
	inputScanner.Scan()
	textLine := inputScanner.Text()
	numPoints, error := strconv.Atoi(strings.Fields(textLine)[0])
	if error != nil {
		log.Fatalf(fmt.Sprintf("non-int first file line '%s'", textLine))
	}

	pointNum := 0
	for inputScanner.Scan() {
		textLine := inputScanner.Text()
		if numDimensions == 0 {
			numDimensions = len(strings.Fields(textLine)[1:])
			points = contiguous2DFloats(numPoints, numDimensions)
		}
		// range [1:] to skip the first string token, which is the (integer) point/line number
		for dimNum, floatString := range strings.Fields(textLine)[1:] {
			newValue, error := strconv.ParseFloat(floatString, 64)
			if error != nil {
				log.Fatalf(fmt.Sprintf("non-float '%s'", floatString))
			}
			(*points)[pointNum][dimNum] = newValue
		}
		pointNum++
	}

	error = inputScanner.Err()
	if error != nil {
		log.Fatal(error)
	}

	return points
}

// firstCentroids creates an initial set of centroids for clustering by
// (psuedo-) randomly selecting points as initial centroids
func firstCentroids(numCentroids int, points *[][]float64, randomSeed uint64) *[][]float64 {
	numPoints := len(*points)
	numDimensions := len((*points)[0])
	centroids := contiguous2DFloats(numCentroids, numDimensions)

	// define random number generation function, enclosing the provided seed;
	// the math here is taken from the UT Parallel Systems lab so the final
	// results of this implementation can be compared for correctness with
	// those of the lab
	nextRand := randomSeed
	randInt := func() int {
		nextRand = nextRand*1103515245 + 12345
		return int((nextRand / 65536) % 32768)
	}

	// randomly select points from the point set to use as initial
	// centroids; again, this is the same process as the K-means lab
	for centIndex := 0; centIndex < numCentroids; centIndex++ {
		pointIndex := randInt() % numPoints
		for dimIndex := 0; dimIndex < numDimensions; dimIndex++ {
			(*centroids)[centIndex][dimIndex] = (*points)[pointIndex][dimIndex]
		}
	}
	return centroids
}

// contiguous2DFloats returns a pointer to a two-dimensional slice of float64;
// it allocates all numRows*numColumns float64s in a single slice to
// preserve data locality (all floats are contiguous in memory) and then
// assigns sub-slices to the individual rows to create the desired
// two-dimensional structure
// note that all floats are default value (0.0)
func contiguous2DFloats(numRows int, numColumns int) *[][]float64 {
	sliceOfRows := make([][]float64, numRows)
	floats := make([]float64, numRows*numColumns)
	for row := 0; row < numRows; row++ {
		sliceOfRows[row] = floats[row*numColumns : (row+1)*numColumns]
	}
	return &sliceOfRows
}

// getIndexes returns the start (inclusive) and end (exclusive) indexes for
// the worker numbered workerNumber, when there are numTasks number of
// tasks to complete and numWorkers number of workers to complete those
// tasks; the tasks are assigned to workers in amounts that are as
// balanced as possible (no worker will have any more or less than any
// other worker by more than 1)
// this could definitely be inlined in main to save the function call
// and eliminate re-calculating subsetMinSize and numMaxSubsets but
// it's cleaner in its own function
func getIndexes(numTasks int, numWorkers int, workerNumber int) (int, int) {
	subsetMinSize := numTasks / numWorkers
	numMaxSubsets := numTasks % numWorkers
	startIndex := workerNumber * subsetMinSize
	stopIndex := startIndex + subsetMinSize
	if workerNumber >= numMaxSubsets {
		// this is not a max size subset; just shift indexes to account
		//   for the max size subssets that came before
		startIndex += numMaxSubsets
		stopIndex += numMaxSubsets
	} else {
		// this is a max size block; shift and grow
		startIndex += workerNumber
		stopIndex += workerNumber + 1
	}
	return startIndex, stopIndex
}

// calculate the Euclidean distance (2-norm) between two vectors;
// since this value is only used for (magnitude) comparison
// purposes with other distances it's not necessary to take the
// relatively expensive square root of the sum of the squares
//
// assumes vecOne and vecTwo have the same number of dimensions
func squareOfEuclideanDistance(vecOne []float64, vecTwo []float64) float64 {
	distSquare := 0.0
	for index, dimOne := range vecOne {
		diff := vecTwo[index] - dimOne
		distSquare += (diff * diff)
	}
	return distSquare
}

// vectorsConverged tests if the square of the Euclidean distance
// between individual corrensponding centroids of each set differ
// by no more than the threshold
//
// assumes the vector sets are the same size and that every vector
// has the same number of dimensions
func vectorsConverged(vecSliceOne [][]float64, vecSliceTwo [][]float64, threshold float64) bool {
	for vecSliceIndex, vecOne := range vecSliceOne {
		if squareOfEuclideanDistance(vecOne, vecSliceTwo[vecSliceIndex]) > threshold {
			return false
		}
	}
	return true
}
