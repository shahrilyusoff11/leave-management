import axios from 'axios';

async function run() {
    try {
        const res = await axios.post('http://localhost:5173/api/v1/login', {
            email: 'superadmin@example.com',
            password: 'P@ssw0rd'
        });
        const token = res.data.token;
        const res2 = await axios.get('http://localhost:5173/api/v1/leave-balance', {
            headers: { Authorization: 'Bearer ' + token }
        });
        console.log('Leave balance status:', res2.status);
    } catch(err) {
        console.error('Error status:', err.response ? err.response.status : err.message);
        console.error('Error data:', err.response ? err.response.data : '');
    }
}
run();
