package com.example.service;

import com.example.core.model.User;
import com.example.core.service.UserService;
import com.example.common.utils.StringUtils;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class UserRegistrationService {
    
    private final UserService userService;
    private final EmailService emailService;
    
    public UserRegistrationService(UserService userService, EmailService emailService) {
        this.userService = userService;
        this.emailService = emailService;
    }
    
    @Transactional
    public User registerUser(String username, String email, String password) {
        // 验证输入参数
        if (StringUtils.isEmpty(username)) {
            throw new IllegalArgumentException("Username is required");
        }
        
        if (!StringUtils.isValidEmail(email)) {
            throw new IllegalArgumentException("Invalid email format");
        }
        
        if (StringUtils.isEmpty(password)) {
            throw new IllegalArgumentException("Password is required");
        }
        
        // 创建用户
        User user = userService.createUser(username, email, password);
        
        // 发送欢迎邮件
        emailService.sendWelcomeEmail(user);
        
        return user;
    }
    
    @Transactional
    public boolean initiatePasswordReset(String email) {
        if (!StringUtils.isValidEmail(email)) {
            throw new IllegalArgumentException("Invalid email format");
        }
        
        return userService.findAllActiveUsers().stream()
                .filter(user -> email.equalsIgnoreCase(user.getEmail()))
                .findFirst()
                .map(user -> {
                    String resetToken = generateResetToken();
                    emailService.sendPasswordResetEmail(user, resetToken);
                    return true;
                })
                .orElse(false);
    }
    
    private String generateResetToken() {
        return "RESET-" + System.currentTimeMillis() + "-" + (int)(Math.random() * 10000);
    }
}