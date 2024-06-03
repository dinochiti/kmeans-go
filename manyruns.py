#!/usr/bin/env python3
import os
import re
import sys
from subprocess import check_output
from time import sleep

#
#  run many times with output formatted for easy graphing
#

if len(sys.argv) != 2:
    print("{} takes a single command line argument for number times to run each variant".format(sys.argv[0]))
    exit()

ITERATIONS = int(sys.argv[1]) + 2

NUM_WORKERS = [" 1 ", " 2 ", " 4 ", " 8 ", " 16 "]
INPUTS = [" ./input/random-n2048-d16-c16.txt ", " ./input/random-n16384-d24-c16.txt ", " ./input/random-n65536-d32-c16.txt "]
INP_NAMES = ["16 dims, 2048 pts", "24 dims, 16384 pts", "32 dims, 65536 pts"]

total_data = [["number of workers", "1", "2", "4", "8", "16"]]
iter_data  = [["number of workers", "1", "2", "4", "8", "16"]]
conv_data = [["number of workers", "1", "2", "4", "8", "16"]]

for input_num in range(len(INPUTS)):
    new_total_row = [INP_NAMES[input_num]]
    total_data.append(new_total_row)
    new_iter_row = [INP_NAMES[input_num]]
    iter_data.append(new_iter_row)
    new_conv_row = [INP_NAMES[input_num]]
    conv_data.append(new_conv_row)
    for num_workers in NUM_WORKERS:
        cmd = "./kmeans -input {} -clusters 16 -seed 8675309 -centroids -threshold 0.0000000001 -workers {}".format(INPUTS[input_num], num_workers)
        print(cmd)
        total_time = 0.0
        total_times = []
        per_iter_time = 0.0
        per_iter_times = []
        conv_num = 0
        for iteration in range(ITERATIONS):
            out = check_output(cmd, shell=True).decode("ascii")
            l = re.search("Total time \(ms\):  (.*)", out)
            m = re.search("iteration \(ms\):  (.*)", out)
            k = re.search("convergence:  (.*) ;", out)
            if l is not None and m is not None:
                per_iter_times.append(float(m.group(1)))
                total_times.append(float(l.group(1)))
                if conv_num == 0 or int(k.group(1)) is conv_num:
                    conv_num = int(k.group(1))
                else:
                    print("Inconsistent convergence!\n")
            else:
                print("Error! Unexpected output format\n")
                print(out, "\n")
                exit()
        total_times.remove(max(total_times))
        total_times.remove(min(total_times))
        total_time = sum(total_times)
        total_time /= ITERATIONS
        new_total_row.append("{:.4f}".format(total_time))
        per_iter_times.remove(max(per_iter_times))
        per_iter_times.remove(min(per_iter_times))
        per_iter_time = sum(per_iter_times)
        per_iter_time /= ITERATIONS
        new_iter_row.append("{:.4f}".format(per_iter_time))
        new_conv_row.append("{}".format(conv_num))

print("\ntime per iteration (ms)\n")
for row in iter_data:
    print("\t".join(row))

print("\ntotal time (ms)\n")
for row in total_data:
    print("\t".join(row))

print("\niterations to converge\n")
for row in conv_data:
    print("\t".join(row))