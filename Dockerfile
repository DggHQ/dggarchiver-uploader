FROM golang:alpine as builder
ARG TARGETARCH
LABEL builder=true multistage_tag="dggarchiver-uploader-builder"
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=${TARGETARCH} go build

FROM alpine:3.17
WORKDIR /app
COPY --from=builder /app/dggarchiver-uploader .
ENTRYPOINT [ "./dggarchiver-uploader" ]