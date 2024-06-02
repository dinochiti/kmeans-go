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

## Inspiration

The UT CS380P Parallel Systems class had 5 labs each solving a different embarrassingly parallelizable challenge using a different language or tool.

We solved clustering with K-Means using C++ and Cuda. (I don't have that repo public here in GitHub because it would serve as a cheat source for future students in the class.)

We also solved equivalence comparison for Binary Search Trees using Go.

I enjoyed Go a lot and decided to cross-solve K-Means with it as a fun exercise.
