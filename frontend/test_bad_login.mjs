import axios from 'axios';

async function run() {
    try {
        const res = await axios.post('http://localhost:5173/api/v1/login', {
            email: 'superadmin@example.com',
            password: 'wrongpassword'
        });
        console.log('Login success:', res.data);
    } catch(err) {
        console.log('Error Status:', err.response ? err.response.status : err.message);
        console.log('Error Data:', err.response ? err.response.data : '');
    }
}
run();
