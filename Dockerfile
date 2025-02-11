FROM golang:1.23-alpine AS builder
WORKDIR /financial_assistant_gau
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
WORKDIR /financial_assistant_gau
COPY --from=builder /financial_assistant_gau/main .
EXPOSE 8901
CMD ["./main"]