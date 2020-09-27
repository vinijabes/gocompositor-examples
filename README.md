docker build -t gocompositor-examples .
docker run -v $(pwd):/gocompositor-examples --env SDP=$SDP -it gocompositor-examples examples/filewebrtc/main.go