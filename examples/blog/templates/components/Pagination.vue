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
  border-top: 3px solid #1a1a1a;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.8rem;
}

.page-link {
  color: #1a1a1a;
  text-decoration: none;
  padding: 0.35rem 0.8rem;
  border: 2px solid #1a1a1a;
  font-weight: 700;
  box-shadow: 3px 3px 0 #1a1a1a;
  transition: transform 0.1s ease, box-shadow 0.1s ease, background 0.1s ease;
}

.page-link:hover {
  text-decoration: none;
  background: #ffd43b;
  transform: translate(-2px, -2px);
  box-shadow: 5px 5px 0 #1a1a1a;
}

.page-disabled {
  color: #bbb;
  border-color: #ccc;
  box-shadow: 3px 3px 0 #ccc;
  cursor: default;
}

.page-disabled:hover {
  background: none;
  color: #bbb;
  transform: none;
  box-shadow: 3px 3px 0 #ccc;
}

.page-info {
  color: #888;
  font-weight: 700;
}
</style>
