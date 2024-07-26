// model to support html fragment loading
function fragment(url) {
    return {
        content: '',
        load() {
            fetch(url)
                .then(response => {
                    if (!response.ok) {
                            throw new Error(`HTTP error status: ${response.status}`);
                    }
                    return response.text()
                })
                .then(text => {
                    this.content = text;
                })
                .catch(error => {
                    console.error(`Error loading fragment ${url}`, error);
                });
        }
    }
}