# Distributed implementation of MapReduce

A distributed implementation of the [MapReduce](http://static.googleusercontent.com/media/research.google.com/en//archive/mapreduce-osdi04.pdf) programming model in GoLang using a word count application as an example. The term "Coordinator" has been used in the implementation instead of the paper's "Master".

# Go setup
Follow the [go installation](https://go.dev/doc/install) guide and check if its correctly installed using ```go version```

# src/main directory
It contains ```mrcoordinator.go``` (the main program that creates a coordinator), ```mrworker.go``` (the main program that creates a worker) & text files to be used for word count.

# src/mr directory
It contains the distributed part of our implementation in the form of ```coordinator.go```, ```worker.go``` & ```rpc.go```

# src/mrapps directory
```wc.go``` is the word count application containing the ```Map()``` & ```Reduce()``` function calls

# Running MapReduce
- Open 2 separate terminals
- Naviagte to the ```src/main``` directory in both
- Build the word count plugin using ```go build -buildmode=plugin ../mrapps/wc.go```
- Run the Coordinator using ```go run mrcoordinator.go pg*.txt```
- In the 2nd terminal, run the worker (you may run more than 1 worker in separate terminals) using ```go run mrworker.go wc.so ```
- Files ```mr-{0-7}-{0-9}``` in ```src/main``` will have all Key/Value pairs with the word as key and its occurence as value
- Files ```mr-out-{0-9}``` contain the output with a list of word followed by the total number of occurences
