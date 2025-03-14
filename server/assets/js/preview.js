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

const toaster = new Notyf({dismissible: true, duration: 2000});

/**
 * Shows the popup modal for an image preview.
 * @param {string} name
 * @param {string} id
 * @param {string} ext
 * @param {string} mimeType
 * @param {number} uploadedAt
 * @return {boolean}
 */
function showImagePreview(name, id, ext, mimeType, uploadedAt) {
    // Ensure all the required elements are created and exist.
    if (!popupElement) return false;
    if (!titleElement) return false;
    const targetUrl = "/f/"+id+ext;

    titleElement.innerText = name;
    previewContainer.src = targetUrl;

    popupElement.showModal();
    return true;
}

window.onload = function() {
    popupElement = document.getElementById("upload-preview-modal");
    titleElement = document.getElementById("upload-preview-modal-title");
    previewContainer = document.getElementById("upload-preview-modal-preview-container");
    const res = showImagePreview("Amazing Image!", "GFORxwmAsPQR", ".png", "image/png", 1741908775);
    console.log(res);
}