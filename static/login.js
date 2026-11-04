(async () => {
    const form = document.getElementById('login-form');
    const errorEl = document.getElementById('login-error');
    const password = document.getElementById('password');
    const confirm = document.getElementById('confirm');
    const btn = document.getElementById('login-btn');

    const { needsSetup } = await (await fetch('/api/auth/status')).json();
    if (needsSetup) {
        document.getElementById('login-title').textContent = 'Create account';
        document.getElementById('login-subtitle').textContent = 'First run: choose the username and password for this server.';
        document.getElementById('confirm-group').classList.remove('hidden');
        password.autocomplete = 'new-password';
        password.minLength = 8;
        confirm.required = true;
        btn.textContent = 'Create account';
    }

    const showError = (msg) => {
        errorEl.textContent = msg;
        errorEl.classList.remove('hidden');
    };

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        errorEl.classList.add('hidden');
        if (needsSetup && password.value !== confirm.value) {
            return showError('Passwords do not match.');
        }
        btn.disabled = true;
        try {
            const res = await fetch(needsSetup ? '/api/auth/setup' : '/api/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username: document.getElementById('username').value, password: password.value }),
            });
            if (res.ok) {
                location.href = '/';
                return;
            }
            const data = await res.json().catch(() => ({}));
            showError(data.error || 'Login failed.');
        } catch {
            showError('Could not reach the server.');
        } finally {
            btn.disabled = false;
        }
    });
})();
