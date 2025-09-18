package com.example.core.service;

import com.example.core.model.User;
import com.example.core.repository.UserRepository;
import com.example.common.utils.StringUtils;
import org.springframework.stereotype.Service;
import java.util.List;
import java.util.Optional;

@Service
public class UserService {
    
    private final UserRepository userRepository;
    
    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }
    
    public User createUser(String username, String email, String password) {
        if (StringUtils.isEmpty(username)) {
            throw new IllegalArgumentException("Username cannot be empty");
        }
        
        if (!StringUtils.isValidEmail(email)) {
            throw new IllegalArgumentException("Invalid email format");
        }
        
        User user = new User();
        user.setUsername(username);
        user.setEmail(email);
        user.setPassword(password);
        user.setStatus(User.UserStatus.ACTIVE);
        
        return userRepository.save(user);
    }
    
    public Optional<User> findUserById(Long id) {
        return userRepository.findById(id);
    }
    
    public List<User> findAllActiveUsers() {
        return userRepository.findByStatus(User.UserStatus.ACTIVE);
    }
    
    public User updateUserStatus(Long userId, User.UserStatus newStatus) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + userId));
        
        user.setStatus(newStatus);
        return userRepository.save(user);
    }
    
    public boolean deleteUser(Long userId) {
        return userRepository.findById(userId)
                .map(user -> {
                    user.setStatus(User.UserStatus.INACTIVE);
                    userRepository.save(user);
                    return true;
                })
                .orElse(false);
    }
    
    public boolean validateUserCredentials(String email, String password) {
        return userRepository.findByEmail(email)
                .filter(user -> user.isActive())
                .filter(user -> user.getPassword().equals(password))
                .isPresent();
    }
}