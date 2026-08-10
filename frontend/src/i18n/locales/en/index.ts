import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import images from './images'

export default {
	...landing,
	...common,
	...dashboard,
	...images,
	...channelMonitorV2,
	...batchImage,
  admin,
  ...misc,
}
