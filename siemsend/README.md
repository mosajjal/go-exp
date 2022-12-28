# SIEMSend
UNIX philosophy inspired SIEM connector. 


The binary work is very simple: get a stream of JSON lines from stdin, send them in batches to a SIEM. Any time a batch fails, the batch will be sent to `stdout`, so any error control and/or backup solution can be piped from `siemsend`

Example:

```sh
tail -F myjsonllogs.json | ./siemsend sentinel --customer_id=yourcustomerid --shared_key=yoursharedkey --log_type=yourlogtype | tee -a failedtosend.json
```

Currently, only Microsoft Sentinel is implemented. More to come if this is popular enough :) 