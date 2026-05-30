/**
 * useCatalog — the model catalog from the BFF (`/api/catalog`). Each entry carries the Exascale
 * pricing extension (modality, credit_type, unit, fixed-point price) the UI renders.
 */
export interface CatalogModel {
  id: string
  object: string
  owned_by: string
  exascale: { modality: string; credit_type: string; unit: string; price: string }
}

export function useCatalog() {
  const models = useState<CatalogModel[]>('catalog:models', () => [])

  /** load fetches the catalog and caches it. */
  async function load() {
    const r = await $fetch<{ object: string; data: CatalogModel[] }>('/api/catalog')
    models.value = r.data
    return models.value
  }

  return { models, load }
}
