<template>
  <nav class="pagination" v-if="pagination.TotalPages > 1">
    <a v-if="pagination.PrevURL !== ''" :href="pagination.PrevURL" class="page-link page-prev">&larr; Previous</a>
    <span v-else class="page-link page-prev page-disabled">&larr; Previous</span>
    <span class="page-info">{{ pagination.Page }} / {{ pagination.TotalPages }}</span>
    <a v-if="pagination.NextURL !== ''" :href="pagination.NextURL" class="page-link page-next">Next &rarr;</a>
    <span v-else class="page-link page-next page-disabled">Next &rarr;</span>
  </nav>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 2.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid #e0ddd8;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.8rem;
}

.page-link {
  color: #b5451b;
  text-decoration: none;
  padding: 0.3rem 0.6rem;
  border: 1px solid currentColor;
  border-radius: 3px;
}

.page-link:hover {
  background: #b5451b;
  color: #fff;
}

.page-disabled {
  color: #ccc;
  border-color: #e0ddd8;
  cursor: default;
}

.page-disabled:hover {
  background: none;
  color: #ccc;
}

.page-info {
  color: #888;
}
</style>
