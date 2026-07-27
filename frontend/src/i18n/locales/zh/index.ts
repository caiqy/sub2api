import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import images from './images'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...images,
  ...batchImage,
  admin,
  ...misc,
}
