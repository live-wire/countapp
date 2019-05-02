#!/bin/bash

for i in {5000..5010}; do if [[ $(lsof -t -i :$i) -eq 0 ]]; then :; else kill -9 $(lsof -t -i :$i); fi; done