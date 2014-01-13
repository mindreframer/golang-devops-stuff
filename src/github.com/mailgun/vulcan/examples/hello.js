function handle(request) {
    var response = get("http://localhost:5000/auth");
    info("Hello from javascript, folks: %v", response);
    if (response.code != 200) {
        return response;
    }
    var forward = {
        rates: {
            "token": ["3 requests/minute", "100 KB/minute"]
        },
        upstreams: ["http://localhost:5000", "http://localhost:5000"]
    };
    return forward;
}

function handleError(request, error) {
    return error
}
