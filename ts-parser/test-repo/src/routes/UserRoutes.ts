import { Router } from 'express';
import { UserController } from '@/controllers/UserController';
import { AuthMiddleware } from '@middleware/AuthMiddleware';
import { validateRequest } from '@middleware/Validation';

const router = Router();
const userController = new UserController();

// Validation schemas
const createUserSchema = {
  email: { required: true, type: 'string', pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/ },
  password: { required: true, type: 'string', minLength: 6 },
  name: { required: true, type: 'string', minLength: 2, maxLength: 50 },
  age: { type: 'number', minLength: 0, maxLength: 150 }
};

const updateUserSchema = {
  email: { type: 'string', pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/ },
  name: { type: 'string', minLength: 2, maxLength: 50 },
  age: { type: 'number', minLength: 0, maxLength: 150 }
};

// Routes
router.get('/', AuthMiddleware.authenticate, userController.getAllUsers.bind(userController));
router.get('/:id', AuthMiddleware.authenticate, userController.getUser.bind(userController));
router.post('/', 
  validateRequest(createUserSchema), 
  userController.createUser.bind(userController)
);
router.put('/:id', 
  AuthMiddleware.authenticate,
  validateRequest(updateUserSchema),
  userController.updateUser.bind(userController)
);
router.delete('/:id', 
  AuthMiddleware.authenticate,
  AuthMiddleware.authorize(['admin']),
  userController.deleteUser.bind(userController)
);

export { router as UserRoutes };