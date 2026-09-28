/** GET /api/supply/sources/:id/attestation — each attestation layer's latest result (F19). */
import { defineEventHandler } from 'h3'
import { proxyJson } from '../../../../utils/api'
import { sourceId } from '../../../../utils/supply'

export default defineEventHandler(event => proxyJson(event, 'compute', `/v1/supply/sources/${sourceId(event)}/attestation`))
