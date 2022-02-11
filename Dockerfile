FROM alpine:3.15
COPY  ./ec2-price-exporter /bin
ENTRYPOINT ["/bin/ec2-price-exporter"]
