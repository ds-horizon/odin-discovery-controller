# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM ${ARCH}gcr.io/distroless/static:nonroot
WORKDIR /
COPY odin-discovery-controller /odin-discovery-controller
USER 65532:65532
CMD ["/odin-discovery-controller"]
