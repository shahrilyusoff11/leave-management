const axios = require('axios');

async function run() {
    try {
        const res = await axios.post('http://localhost:5173/api/v1/login', {
            email: 'superadmin@example.com',
            password: 'P@ssw0rd'
        });
        console.log('Login success:', res.data.user.email);
        
        const token = res.data.token;
        const res2 = await axios.get('http://localhost:5173/api/v1/profile', {
            headers: { Authorization: 'Bearer ' + token }
        });
        console.log('Profile success:', res2.data.email);
    } catch(err) {
        console.error('Error:', err.response ? err.response.data : err.message);
    }
}
run();
