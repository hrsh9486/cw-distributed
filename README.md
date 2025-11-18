Hello examiners!

In order for you to run our program locally, please follow the steps provided below

- Run ```go run broker/broker.go -remote 1```
- Run ``` go run worker/worker.go -remote 1```
- For any subsequent workers that you run, please include the ```-remote 1``` flag, and also specify the ```-port``` flag (default port for the first worker is 8030)
- The ```-remote``` flag is used to configure our main broker and worker functions to run on a local machine, since, to work on a AWS machine, we have configured them to use AWS private IPs, which can only be accessed if on an EC2 instance.

- Once you have broker and worker running, you can run ```go run .```


Many thanks,<br>
Harish and Rishi