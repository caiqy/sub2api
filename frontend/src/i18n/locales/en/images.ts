export default {
  images: {
    badge: 'AI Images',
    title: 'AI Images',
    description: 'Generate, edit, and review AI image work in one place.',
    tabs: { ariaLabel: 'AI image workspace tabs', generate: 'Generate', edit: 'Edit', history: 'History' },
    keySelector: {
      label: 'API Key', loading: 'Loading API keys...', placeholder: 'Select an API key', empty: 'No API keys available yet',
      count: '{count} API keys loaded on this page', pageHint: 'Selection uses the first API key from the current page only.',
      loadFailed: 'Failed to load API keys.', retry: 'Retry'
    },
    panels: {
      generate: { title: 'Generate', description: 'Create a fresh image from a prompt and standard gateway parameters.' },
      edit: { title: 'Edit', description: 'Upload a source image, optionally add a mask, and submit multipart edits.' },
      history: { title: 'History', description: 'Review past image requests and replay their parameters.' }
    },
    forms: {
      generate: {
        prompt: 'Prompt', promptPlaceholder: 'Describe the image you want to create', model: 'Model', size: 'Size',
        sizeHint: 'Official popular presets are shown here. GPT Image 2 also supports auto and more custom sizes that satisfy OpenAI constraints.',
        customSize: 'Custom size', customSizePlaceholder: 'e.g. 2048x1152',
        customSizeRequirements: 'Requirements: multiple of 16, max 3840, aspect ratio <= 3:1',
        customSizeRequired: 'Custom size is required.', customSizeFormat: 'Custom size must use the WIDTHxHEIGHT format, for example 2048x1152.',
        customSizeMultipleOf16: 'Custom size width and height must both be multiples of 16.',
        customSizeMaxEdge: 'Custom size cannot exceed 3840px on either edge.', customSizeAspectRatio: 'Custom size aspect ratio cannot exceed 3:1.',
        customSizePixelRange: 'Custom size pixel count must stay between 655360 and 8294400.',
        quality: 'Quality', background: 'Background', outputFormat: 'Output format', moderation: 'Moderation', n: 'Images',
        submit: 'Generate image', submitting: 'Generating...', submittingWithSeconds: 'Generating... {seconds}s',
        apiKeyRequired: 'Select an API key before submitting.', promptRequired: 'Prompt is required.'
      },
      edit: {
        sourceImage: 'Source image', sourceImageHint: 'PNG, WEBP, or JPEG.', sourceImageInvalid: 'Source image must be an image file.',
        maskImage: 'Mask image', maskImageHint: 'Transparent areas mark editable regions.', sourceImageRequired: 'Source image is required.',
        submit: 'Edit image', submitting: 'Editing...', submittingWithSeconds: 'Editing... {seconds}s'
      }
    },
    results: {
      title: 'Results', description: 'Latest gateway response previews render here.', loading: 'Loading latest result...',
      empty: 'Submit a generate or edit request to see results.', errorTitle: 'Request failed', openPreview: 'Open preview',
      download: 'Download', previewTitle: 'Preview', closePreview: 'Close preview', revisedPrompt: 'Revised prompt', duration: 'Duration'
    },
    history: {
      listTitle: 'Recent requests', empty: 'No image history yet.', loading: 'Loading image history...', loadFailed: 'Failed to load image history.', retry: 'Retry',
      detailTitle: 'History detail', detailEmpty: 'Select a history record to inspect it.', detailLoading: 'Loading history detail...', detailLoadFailed: 'Failed to load history detail.',
      prompt: 'Prompt', noPrompt: 'No prompt', parameters: 'Parameters', images: 'Images', status: 'Status', apiKey: 'API key', createdAt: 'Created at',
      duration: 'Duration', count: 'Images', errorMessage: 'Error message', replay: 'Replay with these settings',
      replayEditNotice: 'Replay restored the edit parameters. Re-upload the source image and optional mask before submitting again.',
      booleanYes: 'Yes', booleanNo: 'No', hadSourceImage: 'Source image uploaded', hadMask: 'Mask uploaded',
      modes: { generate: 'Generate', edit: 'Edit' }, statuses: { success: 'Success', error: 'Error' }
    }
  }
}
