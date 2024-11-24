const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();

// Use CORS middleware
app.use(cors());

app.use(express.static(path.join(__dirname, 'public')));

// Start server
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => console.log(`Server running on port ${PORT}`));
