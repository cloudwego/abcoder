package com.example.service;

import com.example.common.utils.StringUtils;
import com.example.core.model.User;
import org.springframework.stereotype.Service;

@Service
public class EmailService {
    
    public void sendWelcomeEmail(User user) {
        if (user == null || !StringUtils.isValidEmail(user.getEmail())) {
            throw new IllegalArgumentException("Invalid user or email");
        }
        
        String subject = "Welcome to our platform, " + StringUtils.capitalize(user.getUsername());
        String body = String.format(
            "Dear %s,\n\nWelcome to our platform! Your account has been successfully created.\n\nBest regards,\nThe Team",
            StringUtils.capitalize(user.getUsername())
        );
        
        // 模拟发送邮件
        System.out.println("Sending email to: " + user.getEmail());
        System.out.println("Subject: " + subject);
        System.out.println("Body: " + body);
    }
    
    public void sendPasswordResetEmail(User user, String resetToken) {
        if (user == null || !StringUtils.isValidEmail(user.getEmail())) {
            throw new IllegalArgumentException("Invalid user or email");
        }
        
        if (StringUtils.isEmpty(resetToken)) {
            throw new IllegalArgumentException("Reset token cannot be empty");
        }
        
        String subject = "Password Reset Request";
        String body = String.format(
            "Dear %s,\n\nYou have requested a password reset. Please use the following token: %s\n\nThis token will expire in 1 hour.\n\nBest regards,\nThe Team",
            StringUtils.capitalize(user.getUsername()),
            resetToken
        );
        
        // 模拟发送邮件
        System.out.println("Sending password reset email to: " + user.getEmail());
        System.out.println("Subject: " + subject);
        System.out.println("Body: " + body);
    }
}