FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o /payment-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /payment-api /payment-api
EXPOSE 8080
ENTRYPOINT ["/payment-api"]
