const apiKeyInput = document.getElementById("api-key");
const storedApiKey = localStorage.getItem("api-key");
console.log("Stored API key:", storedApiKey);
if (storedApiKey) {
    apiKeyInput.value = storedApiKey;
}

function updateProgressBar(width) {
    const progressBarFill = document.getElementById("progress-bar-fill");
    progressBarFill.style.width = width + "%";
}

async function saveApiKey() {
    const apiKey = apiKeyInput.value;
    console.log("Saving API key:", apiKey);
    localStorage.setItem("api-key", apiKey);

    await updateApiKeyStatus();
}
async function updateApiKeyStatus() {
    const apiKeyStatus = document.getElementById("api-key-status");
    const apiKey = apiKeyInput.value;

    if (apiKey) {
        const isValid = await isApiKeyValid(apiKey);
        apiKeyStatus.style.backgroundColor = isValid ? "green" : "red";
    } else {
        apiKeyStatus.style.backgroundColor = "transparent";
    }
}


async function fetchRustScanResults() {
    const apiKey = apiKeyInput.value;
    const rustScanArgs = document.getElementById("rustscan-args").value;

    if (!apiKey) {
        displayErrorInTerminal("API key is missing. Please enter an API key.");
        return;
    }

    if (!rustScanArgs) {
        displayErrorInTerminal("RustScan arguments are missing. Please enter RustScan arguments.");
        return;
    }

    // Pass the API key and RustScan arguments as query parameters
    const url = new URL("/rustscan", window.location.origin);
    url.searchParams.set("api_key", apiKey);
    url.searchParams.set("args", rustScanArgs);

    // Show the loading spinner and initialize the progress bar
    const loadingSpinner = document.getElementById("loading-spinner");
    loadingSpinner.style.display = "block";
    let progress = 0;
    updateProgressBar(progress);

    // Periodically update the progress bar while RustScan is running
    const progressBarInterval = setInterval(() => {
        progress += 5;
        if (progress > 100) {
            progress = 0;
        }
        updateProgressBar(progress);
    }, 100);

    // Clear the terminal
    const terminal = document.querySelector(".terminal");
    terminal.innerHTML = "";

    // Display the RustScan command in the terminal
    const rustScanCommand = document.createElement("p");
    rustScanCommand.textContent = `$ rustscan ${rustScanArgs}`;
    rustScanCommand.classList.add("command"); // Add a class to the element for styling
    terminal.appendChild(rustScanCommand);

    const response = await fetch(url);
    if (!response.ok) {
        const errorText = await response.text();
        displayErrorInTerminal(`Error: ${errorText}`);
        return;
    }
    const responseText = await response.text();
    const rustScanOutput = JSON.parse(responseText);
    const text = rustScanOutput.join("\n"); // Join the array elements with a line break
    const lines = text.split("\n"); // Split the text into separate lines
    lines.forEach((line) => {
        const lineElement = document.createElement("p");
        lineElement.innerHTML = highlightLine(line); // Highlight the line and set the HTML content of the element
        terminal.appendChild(lineElement);
    });

    // Hide the loading spinner and stop updating the progress bar
    clearInterval(progressBarInterval);
    loadingSpinner.style.display = "none";
    updateProgressBar(0);
}
async function isApiKeyValid(apiKey) {
    const url = new URL("/rustscan", window.location.origin);
    url.searchParams.set("api_key", apiKey);

    const response = await fetch(url, { method: "HEAD" });
    return response.ok;
}

function highlightLine(line) {
    if (line.startsWith("$ rustscan")) {
        return `<span class="command">${line}</span>`;
    } else if (line.startsWith("Open")) {
        return `<span class="open">${line}</span>`;
    } else if (line.includes("open ") && line.includes("/tcp")) {
        return line.replace(/(\d+\/tcp)\s+(\w+)\s+(\w+)/, '<span class="port">$1</span> <span class="state">$2</span> <span class="service">$3</span>');
    } else {
        return `$ ${line}`;
    }
}
function displayErrorInTerminal(errorText) {
    const terminal = document.querySelector(".terminal");
    const errorElement = document.createElement("p");
    errorElement.textContent = errorText;
    errorElement.style.color = "red";
    terminal.appendChild(errorElement);
}