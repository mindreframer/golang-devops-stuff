function handle(request) {
    return {
        upstreams: discover("/api")
    };
}

function handleError(request, error) {
    return error
}
