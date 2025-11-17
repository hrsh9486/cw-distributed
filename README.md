# CSA Coursework: Game of Life (Go)

All documentation is available [here](https://uob-csa.github.io/gol-docs/)

## Setup

### Create Virtual Environment
```bash
python3 -m venv venv
```

### Activate Virtual Environment
```bash
source venv/bin/activate
```

### Install Dependencies
```bash
pip install -r requirements.txt
```

## Running Benchmarks

### 1. Run the benchmark tests
*Note: You may need to delete `results.out` and `results.csv` first*
```bash
go test -run ^$ -bench . -benchtime 1x -count 6 | tee results.out
```

### 2. Generate CSV from results
```bash
go run golang.org/x/perf/cmd/benchstat -format csv results.out | tee results.csv
```

### 3. Plot the data
```bash
python3 plot.py
```

The output image will be saved as `benchmark_data.png`