#!/bin/bash

perl -ne '/(\d+)\t[^\t]+\t(.*)/ and print "$1\t$2\n"' < ${1:-/dev/stdin} | sort | uniq | wc -l

