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

const toaster = new Notyf({dismissible: true, duration: 2000});

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
    deleteButton.href = deleteToken;
    openThumbButton.href = "/thumb/"+id+".png"

    popupElement.showModal();
    return true;
}

window.onload = function() {
    popupElement = document.getElementById("upload-preview-modal");
    titleElement = document.getElementById("upload-preview-modal-title");
    previewContainer = document.getElementById("upload-preview-modal-preview-container");
    openUrlButton = document.getElementById("upload-preview-btn-open-url");
    openThumbButton = document.getElementById("upload-preview-btn-open-thumb");
    deleteButton = document.getElementById("upload-preview-btn-delete");

    // const res = showImagePreview("Amazing Image!", "WfLFgTapmfUa", ".png", "image/png", 1741908775);
    console.log(res);
}