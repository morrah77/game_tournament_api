FROM golang:1.9.1

ENV GOPATH=/go
ENV PATH=$PATH:/go/bin
ENV GOPATH=/proj

COPY ./ /proj/
WORKDIR /proj
RUN go install main
