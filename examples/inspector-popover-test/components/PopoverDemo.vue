<template>
  <div class="demo">
    <h1>Inspector + Popover Test</h1>

    <Card title="Main Page Card" description="This card is in the normal page flow." />

    <!-- Popover API (declarative, no JS required to open/close) -->
    <button popovertarget="native-popover" class="btn">Open Popover</button>
    <div id="native-popover" popover class="popover-box">
      <h3>Inside Popover</h3>
      <Card title="Popover Card" description="This card lives inside a native popover." />
      <button popovertarget="native-popover" popovertargetaction="hide" class="btn btn-sm">Close</button>
    </div>

    <!-- Modal dialog (needs JS) -->
    <button id="open-modal-btn" class="btn">Open Modal Dialog</button>
    <dialog id="my-modal" class="modal-box">
      <h3>Inside Modal Dialog</h3>
      <Card title="Modal Card" description="This card lives inside a modal dialog." />
      <button id="close-modal-btn" class="btn btn-sm">Close</button>
    </dialog>

    <script>
      document.getElementById('open-modal-btn').addEventListener('click', function() {
        document.getElementById('my-modal').showModal();
      });
      document.getElementById('close-modal-btn').addEventListener('click', function() {
        document.getElementById('my-modal').close();
      });
    </script>
  </div>
</template>
<style scoped>
.demo { font-family: sans-serif; max-width: 600px; margin: 2rem auto; }
.btn {
  background: #00ADD8; color: #fff; border: none; border-radius: 4px;
  padding: 0.5rem 1rem; cursor: pointer; margin: 0.25rem 0;
}
.btn-sm { padding: 0.25rem 0.75rem; font-size: 0.85rem; }
.popover-box {
  background: #f0f9ff; border: 2px solid #00ADD8; border-radius: 8px;
  padding: 1rem; min-width: 320px;
}
.modal-box {
  background: #fff; border: 2px solid #334155; border-radius: 8px;
  padding: 1.5rem; min-width: 360px;
}
.modal-box::backdrop { background: rgba(0,0,0,0.5); }
</style>
