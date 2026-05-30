/**
 * GET /api/catalog — the model catalog. Live: inference-gateway /v1/models (OpenAI-shaped, with the
 * Exascale pricing extension). Mock: a canned list matching that shape.
 */
import { defineEventHandler } from 'h3'
import { isMock, proxyJson } from '../utils/api'

export default defineEventHandler((event) => {
  if (isMock(event)) {
    return {
      object: 'list',
      data: [
        { id: 'llama-3.1-70b', object: 'model', owned_by: 'exascale', exascale: { modality: 'text', credit_type: 'text', unit: '1K tokens', price: '25.000000' } },
        { id: 'llama-3.1-8b', object: 'model', owned_by: 'exascale', exascale: { modality: 'text', credit_type: 'text', unit: '1K tokens', price: '5.000000' } },
        { id: 'whisper-large-v3', object: 'model', owned_by: 'exascale', exascale: { modality: 'speech', credit_type: 'speech', unit: '1 minute', price: '10.000000' } },
      ],
    }
  }
  return proxyJson(event, 'gateway', '/v1/models')
})
