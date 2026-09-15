Hello examiners!

We have uploaded our 2 extensions, halo exchange and fault tolerance as zip files. To run each of them, please unzip the files and follow the instructions below.

For halo exchange:
- Run ```go run broker/broker.go -remote 1```
- Run ``` go run worker/worker.go -remote 1```
- For any subsequent workers that you run, please include the ```-remote 1``` flag, and also specify the ```-port``` flag (default port for the first worker is 8030)
- The ```-remote``` flag is used to configure our main broker and worker functions to run on a local machine, since, to work on a AWS machine, we have configured them to use AWS private IPs, which can only be accessed if on an EC2 instance.
- Once you have broker and worker running, you can run ```go run .```

For fault tolerance:
- Follow the same instructions as halo exchange to set the system up
- As the program runs kill a worker using ```ctrl+C```, the system should still keep running
- You can also run ```go test ./tests -v -run TestGol```, to see how the system handles the tests
- Debug statements are included to demonstrate fault tolerance by showing the points where a worker fails and displaying the size of the world each worker is processing

Many thanks,<br>
Harish and Rishi