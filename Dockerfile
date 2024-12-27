# Copyright Contributors to the Open Cluster Management project
FROM golang:1.23 AS builder

ENV DIRPATH /go/src/open-cluster-management.io/clusteradm
WORKDIR ${DIRPATH}

COPY . .

# RUN apt-get update && apt-get install net-tools && make vendor
RUN make build


FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ENV USER_UID=10001

COPY --from=builder /go/src/open-cluster-management.io/clusteradm/bin/clusteradm /

USER ${USER_UID}
