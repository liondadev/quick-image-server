/**
 * @type {HTMLDialogElement}
 */
let popupElement;

/**
 * @type {HTMLSpanElement}
 */
let titleElement;

/**
 * @type {HTMLIFrameElement}
 */
let previewContainer;

/**
 * @type {HTMLAnchorElement}
 */
let openUrlButton;

/**
 * @type {HTMLAnchorElement}
 */
let openThumbButton;

/**
 * @type {HTMLAnchorElement}
 */
let deleteButton;

/**
 * @type {HTMLImageElement}
 */
let closeModalButton;

/**
 * @type {((event: Event) => void) | undefined}
 */
let lastDeleteHandler;

/**
 * @type {HTMLAnchorElement}
 */
let bubbledPngButton;

/**
 * @type {HTMLAnchorElement}
 */
let bubbledGifButton;

/**
 * Shows the popup modal for an image preview.
 * @param {string} name
 * @param {string} id
 * @param {string} ext
 * @param {string} mimeType
 * @param {number} uploadedAt
 * @param {string} deleteToken
 * @return {boolean}
 */
function showImagePreview(name, id, ext, mimeType, uploadedAt, deleteToken) {
    // Ensure all the required elements are created and exist.
    if (!popupElement) return false;
    if (!titleElement) return false;
    if (!openUrlButton) return false;
    if (!openThumbButton) return false;
    if (!deleteButton) return false;

    const targetUrl = "/f/"+id+ext;
    const deleteUrl = "/delete/"+id+"/"+deleteToken;

    titleElement.innerText = name;
    previewContainer.src = targetUrl;

    openUrlButton.href = targetUrl;
    openThumbButton.href = "/thumb/"+id+".png"
    bubbledPngButton.href = "/bubble/"+id+".png"
    bubbledGifButton.href = "/bubble/"+id+".gif"

    // Remove old event handler so we don't delete old files.
    if (lastDeleteHandler)
        deleteButton.removeEventListener("click", lastDeleteHandler);

    // Set the new event handler.
    lastDeleteHandler = async (event) => {
        const shouldDelete = confirm(`Are you sure you want to PERMANENTLY delete ${name}?`);
        if (!shouldDelete) return;

        const status = await fetch(deleteUrl).then((r) => r.status).catch((err) => {
            alert("Failed to delete the file. Please check your JS console.")
            console.error(err)
        })
        if (!status) return; // caught error
        if (status !== 200)
            alert("Failed to delete the file. Received non-200 error code.");

        window.location = window.location; // refresh page
    }
    deleteButton.addEventListener("click", lastDeleteHandler)

    popupElement.showModal();
    return true;
}

// Set the vars to what we expect...
window.onload = function() {
    popupElement = document.getElementById("upload-preview-modal");
    titleElement = document.getElementById("upload-preview-modal-title");
    previewContainer = document.getElementById("upload-preview-modal-preview-container");
    openUrlButton = document.getElementById("upload-preview-btn-open-url");
    openThumbButton = document.getElementById("upload-preview-btn-open-thumb");
    deleteButton = document.getElementById("upload-preview-btn-delete");
    closeModalButton = document.getElementById("upload-preview-close-button")
    bubbledPngButton = document.getElementById("upload-preview-btn-open-bubbled-png");
    bubbledGifButton = document.getElementById("upload-preview-btn-open-bubbled-gif");

    closeModalButton.addEventListener("click", () => {
        if (popupElement.open) popupElement.close();
    })

    if (window.extraload) window.extraload();
}