package com.example.core.model;

import com.example.common.model.BaseEntity;
import com.example.common.utils.StringUtils;

public class User extends BaseEntity {
    
    private String username;
    private String email;
    private String password;
    private UserStatus status;
    
    public enum UserStatus {
        ACTIVE, INACTIVE, SUSPENDED
    }
    
    public String getUsername() {
        return username;
    }
    
    public void setUsername(String username) {
        if (StringUtils.isNotEmpty(username)) {
            this.username = username.trim();
        }
    }
    
    public String getEmail() {
        return email;
    }
    
    public void setEmail(String email) {
        if (StringUtils.isValidEmail(email)) {
            this.email = email.toLowerCase();
        }
    }
    
    public String getPassword() {
        return password;
    }
    
    public void setPassword(String password) {
        if (StringUtils.isNotEmpty(password)) {
            this.password = password;
        }
    }
    
    public UserStatus getStatus() {
        return status;
    }
    
    public void setStatus(UserStatus status) {
        this.status = status;
    }
    
    public boolean isActive() {
        return status == UserStatus.ACTIVE;
    }
    
    @Override
    public String toString() {
        return "User{" +
                "id=" + getId() +
                ", username='" + username + '\'' +
                ", email='" + email + '\'' +
                ", status=" + status +
                '}';
    }
}