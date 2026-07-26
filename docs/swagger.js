// docs/swagger.js
import swaggerJSDoc from 'swagger-jsdoc';

const options = {
  definition: {
    openapi: '3.0.0',
    info: {
      title: 'Ekomasi Implementation API',
      version: '1.0.0',
      description: 'API documentation for Ekomasi implementation project',
    },
    servers: [
      {
        url: 'http://localhost:8032', // You can use an env var if needed
      },
    ],
    components: {
      securitySchemes: {
        bearerAuth: {
          type: 'http',
          scheme: 'bearer',
          bearerFormat: 'JWT',
        },
      },
    },
    security: [
      {
        bearerAuth: [],
      },
    ],
  },
  apis: ['./src/**/*.js'], // Adjust path to match where your routes/controllers live
};

const swaggerSpec = swaggerJSDoc(options);

export default swaggerSpec;
