package com.example.web.controller;

import com.example.core.model.User;
import com.example.service.UserRegistrationService;
import com.example.core.service.UserService;
import com.example.common.utils.StringUtils;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import java.util.List;
import java.util.Optional;

@RestController
@RequestMapping("/api/users")
public class UserController {
    
    private final UserService userService;
    private final UserRegistrationService registrationService;
    
    public UserController(UserService userService, UserRegistrationService registrationService) {
        this.userService = userService;
        this.registrationService = registrationService;
    }
    
    @PostMapping("/register")
    public ResponseEntity<User> registerUser(@RequestBody UserRegistrationRequest request) {
        try {
            User user = registrationService.registerUser(
                    request.getUsername(),
                    request.getEmail(),
                    request.getPassword()
            );
            return ResponseEntity.ok(user);
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().build();
        }
    }
    
    @GetMapping("/{id}")
    public ResponseEntity<User> getUserById(@PathVariable Long id) {
        Optional<User> user = userService.findUserById(id);
        return user.map(ResponseEntity::ok)
                  .orElse(ResponseEntity.notFound().build());
    }
    
    @GetMapping("/active")
    public ResponseEntity<List<User>> getAllActiveUsers() {
        List<User> users = userService.findAllActiveUsers();
        return ResponseEntity.ok(users);
    }
    
    @PutMapping("/{id}/status")
    public ResponseEntity<User> updateUserStatus(
            @PathVariable Long id,
            @RequestParam User.UserStatus status) {
        try {
            User user = userService.updateUserStatus(id, status);
            return ResponseEntity.ok(user);
        } catch (IllegalArgumentException e) {
            return ResponseEntity.notFound().build();
        }
    }
    
    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteUser(@PathVariable Long id) {
        boolean deleted = userService.deleteUser(id);
        return deleted ? ResponseEntity.noContent().build()
                      : ResponseEntity.notFound().build();
    }
    
    @PostMapping("/reset-password")
    public ResponseEntity<Void> resetPassword(@RequestBody PasswordResetRequest request) {
        boolean initiated = registrationService.initiatePasswordReset(request.getEmail());
        return initiated ? ResponseEntity.ok().build()
                        : ResponseEntity.notFound().build();
    }
    
    public static class UserRegistrationRequest {
        private String username;
        private String email;
        private String password;
        
        public String getUsername() { return username; }
        public void setUsername(String username) { this.username = username; }
        
        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }
        
        public String getPassword() { return password; }
        public void setPassword(String password) { this.password = password; }
    }
    
    public static class PasswordResetRequest {
        private String email;
        
        public String getEmail() { return email; }
        public void setEmail(String email) { this.email = email; }
    }
}