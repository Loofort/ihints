#!/bin/bash

perl -ne '/\d+\t[^\t]+\t(.*)/ and print "$1\n"' < ${1:-/dev/stdin} | sort | uniq 


