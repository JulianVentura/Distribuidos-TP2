FROM golang:1.17 AS builder
# Client uses docker multistage builds feature https://docs.docker.com/develop/develop-images/multistage-build/
# First stage is used to compile golang binary and second stage is used to only copy the 
# binary generated to the deploy image. 
# Docker multi stage does not delete intermediate stages used to build our image, so we need 
# to delete it by ourselves. Since docker does not give a good alternative to delete the intermediate images
# we are adding a very specific label to the image to then find these kind of images and delete them
LABEL intermediateStageToBeDeleted=true

RUN mkdir -p /build
WORKDIR /build/

COPY ./go.mod ./go.sum ./
COPY ./vendor ./vendor/
COPY ./common ./common/
COPY ./server/common ./server/common/
COPY ./server/best_sentiment_avg_downloader ./server/best_sentiment_avg_downloader
# CGO_ENABLED must be disabled to run go binary in Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin/best_sentiment_avg_downloader ./server/best_sentiment_avg_downloader

FROM busybox:latest
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/bin/best_sentiment_avg_downloader /downloader
ENTRYPOINT ["/bin/sh"]
