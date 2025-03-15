const buttonElement = document.getElementById("import-button");
const consoleElement = document.getElementById("import-console");
const inputElement = document.getElementById("import-entry");

const handler = () => {
    buttonElement.removeEventListener("click", handler)

    const val = encodeURIComponent(inputElement.value);
    const evt = new EventSource("/import-api?fileName="+val, {withCredentials: true})

    function addMessage(type, content) {
        const el = document.createElement("pre")
        el.classList.add(type)
        el.innerText = `[${type.toLocaleUpperCase()}] ${content}`

        consoleElement.appendChild(el)
        consoleElement.scrollTop = consoleElement.scrollHeight
    }

    evt.onmessage = async (event) => {
        try {
            const data = await JSON.parse(event.data)
            addMessage(data.type, data.content)
        } catch (e) {
            addMessage("fail", "Error while doing import. Please check JS console.")
            console.error(e)
        }
    }

    evt.onerror = function() {
        addMessage("info", "Connection Dropped. Re-Enabling the button...")
        buttonElement.addEventListener("click", handler)
        evt.close();
    };
}

buttonElement.addEventListener("click", handler);
console.log(buttonElement)