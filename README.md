# kmeans

An application to calculate clusters for a set of points using the K-means method.

This is a Go version of a lab exercise from the University of Texas at Austin CS380P Parallel Systems course in the online Masters of Computer Science program.

The points to cluster are read in from an input file; the file format and the example input files used are directly from the lab.

https://www.cs.utexas.edu/~rossbach/cs380p/index.html

## Build

Clone the repo and build from the root directory as usual for Go, i.e., 

```console
you@there:~/kmeans-go$ go build
```

## Timings

Does it work? Does adding goroutines as workers improve performance?

The `manyruns.py` Python script runs the executable many times with different inputs, several times per input set, with a max and min result discarded.

Here's the total time in milliseconds trended across input sets and numbers of worker routines:

| number of workers  |  1     |  2     |  4     |  8     |  16    |
|--------------------|--------|--------|--------|--------|--------|
| 16 dims, 2048 pts  | 7.5455 | 6.3636 | 6.3636 | 6.3636 | 6.8182 |
| 24 dims, 16384 pts | 68.818 | 49.090 | 48.818 | 50.454 | 51.000 |
| 32 dims, 65536 pts | 354.68 | 235.09 | 208.04 | 200.95 | 200.77 |

Clearly more workers usually get the job done faster, especially as the amount of data increases.

## Inspiration

The UT CS380P Parallel Systems class had 5 labs each solving a different embarrassingly parallelizable challenge using a different language or tool.

We solved clustering with K-Means using C and Cuda. (I don't have that repo public here in GitHub because it would serve as a cheat source for future students in the class.)

We also solved equivalence comparison for Binary Search Trees using Go.

I enjoyed Go a lot and decided to cross-solve K-Means with it as a fun exercise.
