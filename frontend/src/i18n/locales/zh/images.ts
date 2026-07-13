export default {
  images: {
    badge: 'AI 生图',
    title: 'AI生图',
    description: '在一个页面中生成、编辑和查看 AI 图片。',
    tabs: { ariaLabel: 'AI 图片工作区标签', generate: '生成', edit: '编辑', history: '历史' },
    keySelector: {
      label: 'API 密钥', loading: '正在加载 API 密钥...', placeholder: '请选择 API 密钥', empty: '暂无可用 API 密钥',
      count: '当前页已加载 {count} 个 API 密钥', pageHint: '仅从当前页选择 API 密钥。', loadFailed: 'API 密钥加载失败。', retry: '重试'
    },
    panels: {
      generate: { title: '生成', description: '通过提示词和标准网关参数生成新图片。' },
      edit: { title: '编辑', description: '上传原图，可选遮罩，并提交图片编辑请求。' },
      history: { title: '历史', description: '查看历史图片请求并复用其参数。' }
    },
    forms: {
      generate: {
        prompt: '提示词', promptPlaceholder: '描述要生成的图片', model: '模型', size: '尺寸',
        sizeHint: '此处列出常用官方尺寸。GPT Image 2 还支持 auto 和满足 OpenAI 限制的自定义尺寸。',
        customSize: '自定义尺寸', customSizePlaceholder: '例如 2048x1152', customSizeRequirements: '要求：边长为 16 的倍数、最大 3840、宽高比不超过 3:1',
        customSizeRequired: '请输入自定义尺寸。', customSizeFormat: '自定义尺寸必须使用 WIDTHxHEIGHT 格式，例如 2048x1152。',
        customSizeMultipleOf16: '自定义尺寸的宽和高都必须是 16 的倍数。', customSizeMaxEdge: '自定义尺寸任一边不能超过 3840px。',
        customSizeAspectRatio: '自定义尺寸宽高比不能超过 3:1。', customSizePixelRange: '自定义尺寸总像素必须在 655360 到 8294400 之间。',
        quality: '质量', background: '背景', outputFormat: '输出格式', moderation: '内容审核', n: '图片数量',
        submit: '生成图片', submitting: '生成中...', submittingWithSeconds: '生成中... {seconds}s',
        apiKeyRequired: '提交前请选择 API 密钥。', promptRequired: '请输入提示词。'
      },
      edit: {
        sourceImage: '原图', sourceImageHint: '支持 PNG、WEBP 或 JPEG。', sourceImageInvalid: '原图必须是图片文件。',
        maskImage: '遮罩图片', maskImageHint: '透明区域表示可编辑范围。', sourceImageRequired: '请上传原图。',
        submit: '编辑图片', submitting: '编辑中...', submittingWithSeconds: '编辑中... {seconds}s'
      }
    },
    results: {
      title: '生成结果', description: '最新网关响应将在这里预览。', loading: '正在加载最新结果...', empty: '提交生成或编辑请求后可查看结果。',
      errorTitle: '请求失败', openPreview: '打开预览', download: '下载', previewTitle: '图片预览', closePreview: '关闭预览', revisedPrompt: '修订后的提示词', duration: '耗时'
    },
    history: {
      listTitle: '最近请求', empty: '暂无图片历史。', loading: '正在加载图片历史...', loadFailed: '图片历史加载失败。', retry: '重试',
      detailTitle: '历史详情', detailEmpty: '请选择一条历史记录。', detailLoading: '正在加载历史详情...', detailLoadFailed: '历史详情加载失败。',
      prompt: '提示词', noPrompt: '无提示词', parameters: '参数', images: '图片', status: '状态', apiKey: 'API 密钥', createdAt: '创建时间',
      duration: '耗时', count: '图片数量', errorMessage: '错误信息', replay: '复用这些设置',
      replayEditNotice: '已恢复编辑参数，请重新上传原图和可选遮罩后提交。', booleanYes: '是', booleanNo: '否', hadSourceImage: '已上传原图', hadMask: '已上传遮罩',
      modes: { generate: '生成', edit: '编辑' }, statuses: { success: '成功', error: '失败' }
    }
  }
}
