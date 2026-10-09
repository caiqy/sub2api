export default {
  images: {
    badge: 'AI Images', title: 'AI Images', description: 'Describe a scene, create an image, and keep editing the result.',
    tabs: { ariaLabel: 'Image creation mode', generate: 'Generate', edit: 'Edit', history: 'History' },
    keySelector: {
      label: 'Platform API key', loading: 'Loading API keys…', placeholder: 'Select an API key', empty: 'No API keys yet. Create one to get started.',
      count: '{count} keys on this page', pageHint: 'Use pagination to select any of your platform API keys.', loadFailed: 'Could not load API keys.', retry: 'Retry'
    },
    models: {
      loading: 'Reading image models for this key…', failed: 'Could not read models.', empty: 'This key returned no GPT Image candidates. Check its group configuration or choose another key.',
      visibilityNotice: 'Models reflect this key’s configuration. Upstream account eligibility still determines actual availability.'
    },
    panels: {
      generate: { title: 'Create an image', description: 'Describe the subject, style, lighting, and composition.' },
      edit: { title: 'Keep editing', description: 'Upload an existing image and describe what to change.' },
      history: { title: 'History', description: 'Review results, reuse settings, or send an output to the editor.' }
    },
    forms: {
      generate: {
        prompt: 'Scene description', promptPlaceholder: 'Describe the subject, style, lighting, and composition…',
        model: 'Model', modelRequired: 'Choose an image model visible to the current key.', size: 'Aspect ratio',
        autoSize: 'Auto', squareSize: 'Square 1:1', landscapeSize: 'Landscape 3:2', portraitSize: 'Portrait 2:3',
        sizeHint: 'Auto lets the model choose the output size. Select a ratio for a fixed composition. Auto may not match your reference dimensions.',
        customSize: 'Custom size', customSizePlaceholder: 'e.g. 2048x1152', customSizeRequirements: 'Edges must be multiples of 16, at most 3840px, with an aspect ratio up to 3:1. High resolutions can take longer.',
        customSizeRequired: 'Enter a custom size.', customSizeFormat: 'Use WIDTHxHEIGHT, for example 2048x1152.',
        customSizeMultipleOf16: 'Width and height must both be multiples of 16.', customSizeMaxEdge: 'Neither edge can exceed 3840px.',
        customSizeAspectRatio: 'The aspect ratio cannot exceed 3:1.', customSizePixelRange: 'Pixel count must be between 655360 and 8294400.',
        quality: 'Quality', background: 'Background', outputFormat: 'Output format', moderation: 'Moderation', n: 'Images',
        advanced: 'Advanced settings', parametersAdjusted: 'Incompatible settings were adjusted for the current model.',
        transparentFormatAdjusted: 'Transparent backgrounds require PNG or WebP. The output format was changed to PNG.',
        outputCompression: 'Compression quality (0–100)', compressionInvalid: 'Compression quality must be an integer from 0 to 100.',
        submit: 'Generate image', submitting: 'Generating…', submittingWithSeconds: 'Generating… {seconds}s',
        apiKeyRequired: 'Select an API key before submitting.', promptRequired: 'Describe the image to create or the changes to make.'
      },
      edit: {
        sourceImage: 'Original / reference images', sourceImageHint: 'Upload the images you want to change. PNG, JPEG, or WebP; up to 16 images, each at most 20 MiB.',
        sourceImageInvalid: 'Choose a real PNG, JPEG, or WebP image whose content matches its file type.', sourceImageRequired: 'Upload at least one original / reference image.',
        sourceImageLimit: 'You can add up to 16 reference images.', sourceImageTooLarge: 'Each reference image must be at most 20 MiB.', sourceImageDecode: 'This image cannot be decoded. Export it again and re-upload.',
        maskImage: 'Area to edit (optional)', maskImageHint: 'Upload a mask to guide the edit area, or leave it empty to edit from your description.',
        maskScopeHint: 'The mask applies to the first reference: transparent areas are intended to change; opaque areas are intended to remain. Use a PNG with alpha, matching dimensions, under 4 MB. This is guidance, not pixel-exact preservation.',
        maskPngRequired: 'The mask must be a PNG image.', maskTooLarge: 'The mask must be smaller than 4,000,000 bytes.', maskAlphaRequired: 'The PNG mask must contain an alpha channel.', maskDimensions: 'The mask dimensions must match the first reference image.',
        removeImage: 'Remove reference', removeMask: 'Remove mask', validating: 'Checking images…',
        submit: 'Edit image', submitting: 'Editing…', submittingWithSeconds: 'Editing… {seconds}s'
      }
    },
    results: {
      title: 'Image preview', description: 'Drafts appear as they arrive. Download or keep editing once complete.', loading: 'Waiting for the image…', empty: 'Describe a scene to preview the result here, or upload an existing image to edit.',
      errorTitle: 'Request incomplete', openPreview: 'Enlarge', download: 'Download', previewTitle: 'Image preview', closePreview: 'Close', revisedPrompt: 'Revised description', duration: 'Duration',
      draft: 'Draft preview · Not complete', stop: 'Stop waiting', stopNotice: 'Stopping only disconnects the browser. The upstream may keep generating and charging. Check results or history before submitting again.',
      sendToEdit: 'Send to edit', editNotice: 'The result is now the first reference image. Describe what to change; nothing is submitted automatically.',
      reading: 'Reading the image for editing…', readFailed: 'The image could not be read. Its link may have expired or be blocked by CORS. Open the original, save it, and re-upload.',
      imageUnavailable: 'This image is unavailable; its link may have expired. Try opening the original or re-uploading a saved copy.', openOriginal: 'Open original',
      states: { idle: 'Ready to create', generating: 'Generating', success: 'Complete', error: 'Incomplete', stopped: 'Stopped waiting' }
    },
    history: {
      listTitle: 'Requests', empty: 'No image history yet.', loading: 'Loading history…', loadFailed: 'Could not load image history.', retry: 'Retry',
      detailTitle: 'History details', detailEmpty: 'Select a record to view its settings and images.', detailLoading: 'Reading details…', detailLoadFailed: 'Could not load history details.',
      prompt: 'Scene description', noPrompt: 'Original description was not retained', parameters: 'Settings', images: 'Images', status: 'Status', apiKey: 'API key', createdAt: 'Created at',
      duration: 'Duration', count: 'Images', errorMessage: 'Error', replay: 'Reuse settings',
      replayEditNotice: 'Edit settings restored. Re-upload the original / reference images and any mask before submitting.', booleanYes: 'Yes', booleanNo: 'No', hadSourceImage: 'Included references', hadMask: 'Included a mask',
      expand: 'Expand', collapse: 'Collapse', refresh: 'Refresh', mode: 'Mode', allKeys: 'All keys', allModes: 'All modes', allStatuses: 'All statuses', keyId: 'Key #{id}',
      summaryUnavailable: 'Summary not retained · Open details', noImages: 'This record has no available final image.',
      detailUnavailable: 'Image details were cleared or not retained. Usage remains in the list, but the original image cannot be recovered from billing records.',
      retentionNotice: 'Image detail retention is configured by the administrator and shared across the site. Usage records do not guarantee permanent images. Reusing edit settings requires uploading the original references and mask again.',
      modes: { generate: 'Generate', edit: 'Edit' }, statuses: { success: 'Success', error: 'Failed' }
    }
  }
}
