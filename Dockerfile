FROM scratch
COPY onepiece /
EXPOSE 8080
ENTRYPOINT ["/onepiece", "web"]