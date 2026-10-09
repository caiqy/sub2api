export default {
  images: {
    badge: 'AI 生图', title: 'AI 生图', description: '描述画面、生成图片，把结果继续修改。',
    tabs: { ariaLabel: '图片创作模式', generate: '生成', edit: '编辑', history: '历史记录' },
    keySelector: {
      label: '平台 API Key', loading: '正在加载 API Key…', placeholder: '请选择 API Key', empty: '暂无 API Key，请先创建一个。',
      count: '本页 {count} 个 API Key', pageHint: '可翻页选择你的平台 API Key。', loadFailed: 'API Key 加载失败。', retry: '重试'
    },
    models: {
      loading: '正在读取这个 Key 的图片模型…', failed: '模型读取失败。', empty: '此 Key 未返回 GPT Image 候选，请检查分组配置或选择其他 Key。',
      visibilityNotice: '模型来自此 Key 的配置；实际可用性仍取决于上游账号与资格。'
    },
    panels: {
      generate: { title: '开始创作', description: '描述你想看到的主体、风格和构图。' },
      edit: { title: '继续修改', description: '上传已有图片，再描述需要怎样修改。' },
      history: { title: '历史记录', description: '查看结果，复用设置或将输出送入编辑。' }
    },
    forms: {
      generate: {
        prompt: '画面描述', promptPlaceholder: '描述主体、风格、光线和构图，例如：一盏米白色台灯，暖光，简洁的产品摄影…',
        model: '模型', modelRequired: '请选择当前 Key 可见的图片模型。', size: '画面比例',
        autoSize: '自动', squareSize: '方形 1:1', landscapeSize: '横向 3:2', portraitSize: '纵向 2:3',
        sizeHint: '自动由模型决定输出尺寸；需要固定构图时选择比例。自动不保证与参考图同尺寸。',
        customSize: '自定义尺寸', customSizePlaceholder: '例如 2048x1152', customSizeRequirements: '边长为 16 的倍数，最大 3840px，宽高比不超过 3:1；高分辨率可能耗时更长。',
        customSizeRequired: '请输入自定义尺寸。', customSizeFormat: '请使用 WIDTHxHEIGHT 格式，例如 2048x1152。',
        customSizeMultipleOf16: '宽和高都必须是 16 的倍数。', customSizeMaxEdge: '任一边不能超过 3840px。',
        customSizeAspectRatio: '宽高比不能超过 3:1。', customSizePixelRange: '总像素必须在 655360 到 8294400 之间。',
        quality: '质量', background: '背景', outputFormat: '输出格式', moderation: '内容审核', n: '图片数量',
        advanced: '高级参数', parametersAdjusted: '已根据当前模型调整不兼容的参数。',
        transparentFormatAdjusted: '透明背景需要 PNG 或 WebP，输出格式已改为 PNG。',
        outputCompression: '压缩质量（0–100）', compressionInvalid: '压缩质量必须是 0 到 100 的整数。',
        submit: '生成图片', submitting: '生成中…', submittingWithSeconds: '生成中… {seconds}s',
        apiKeyRequired: '提交前请选择 API Key。', promptRequired: '请描述要生成或修改的画面。'
      },
      edit: {
        sourceImage: '原始 / 参考图片', sourceImageHint: '上传你想修改的图片。支持 PNG、JPEG、WebP，最多 16 张，每张不超过 20 MiB。',
        sourceImageInvalid: '请选择真实的 PNG、JPEG 或 WebP 图片，文件类型须与内容一致。', sourceImageRequired: '请至少上传一张原始 / 参考图片。',
        sourceImageLimit: '最多添加 16 张参考图。', sourceImageTooLarge: '参考图每张不能超过 20 MiB。', sourceImageDecode: '无法解码这张图片，请重新导出后上传。',
        maskImage: '局部修改范围（可选）', maskImageHint: '上传蒙版，指定主要修改哪里；留空则根据描述编辑。',
        maskScopeHint: '蒙版作用于第一张参考图：透明区域希望修改，不透明区域希望保留。需要同尺寸、带透明通道的 PNG，小于 4 MB；这是引导，不保证逐像素保留。',
        maskPngRequired: '蒙版必须是 PNG 图片。', maskTooLarge: '蒙版必须小于 4,000,000 字节。', maskAlphaRequired: '蒙版 PNG 必须带透明通道。', maskDimensions: '蒙版的像素尺寸必须与第一张参考图一致。',
        removeImage: '移除参考图', removeMask: '移除蒙版', validating: '正在检查图片…',
        submit: '编辑图片', submitting: '编辑中…', submittingWithSeconds: '编辑中… {seconds}s'
      }
    },
    results: {
      title: '图片预览', description: '草稿逐步呈现，完成后可下载或继续编辑。', loading: '正在等待图片…', empty: '填写画面描述，生成后在这里预览。也可以上传已有图片继续修改。',
      errorTitle: '请求未完成', openPreview: '放大', download: '下载', previewTitle: '图片预览', closePreview: '关闭', revisedPrompt: '模型修订描述', duration: '耗时',
      draft: '草稿预览 · 尚未完成', stop: '停止等待', stopNotice: '停止等待仅断开浏览器读取，上游可能继续生成并计费。请先检查结果或历史，避免重复提交。',
      sendToEdit: '送入编辑', editNotice: '结果已作为第一张参考图。请描述需要怎样修改；不会自动提交。',
      reading: '正在读取图片以用于编辑…', readFailed: '图片无法读取，链接可能已失效或受到跨域限制。可以打开原图，保存后重新上传。',
      imageUnavailable: '图片暂不可用，链接可能已失效。请尝试打开原图或重新上传保存的图片。', openOriginal: '打开原图',
      states: { idle: '等待创作', generating: '生成中', success: '已完成', error: '未完成', stopped: '已停止等待' }
    },
    history: {
      listTitle: '请求记录', empty: '暂无图片历史。', loading: '正在加载历史…', loadFailed: '图片历史加载失败。', retry: '重试',
      detailTitle: '历史详情', detailEmpty: '点击记录查看参数和图片。', detailLoading: '正在读取详情…', detailLoadFailed: '历史详情加载失败。',
      prompt: '画面描述', noPrompt: '原始描述未保留', parameters: '参数', images: '图片', status: '状态', apiKey: 'API Key', createdAt: '创建时间',
      duration: '耗时', count: '图片数量', errorMessage: '错误信息', replay: '复用设置',
      replayEditNotice: '编辑参数已恢复，请重新上传原始 / 参考图片及所需蒙版后提交。', booleanYes: '是', booleanNo: '否', hadSourceImage: '包含参考图', hadMask: '包含蒙版',
      expand: '展开', collapse: '收起', refresh: '刷新', mode: '模式', allKeys: '所有 Key', allModes: '所有模式', allStatuses: '所有状态', keyId: 'Key #{id}',
      summaryUnavailable: '摘要未保留 · 点击查看详情', noImages: '此记录没有可用的最终图片。',
      detailUnavailable: '图片详情已被清理或未保留，仍可查看列表中的用量记录；无法从账单恢复原图。',
      retentionNotice: '图片详情按管理员配置保留，额度全站共享；用量记录不代表图片永久可用。编辑记录复用参数后，需要重新上传原始参考图和蒙版。',
      modes: { generate: '生成', edit: '编辑' }, statuses: { success: '成功', error: '失败' }
    }
  }
}
