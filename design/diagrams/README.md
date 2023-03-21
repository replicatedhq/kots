## Build diagrams

Run the following command to build a diagram:

```
docker run --rm -u $(id -u):$(id -g) -v $PWD:/data/ minlag/mermaid-cli:9.4.0 -i /data/<chart name>.mms -o <chart name>.svg
```
